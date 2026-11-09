package app

import (
	"database/sql"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
	"github.com/selfhostit/food"

	"github.com/selfhostit/shoppa/catalogue"
	"github.com/selfhostit/shoppa/products"
)

// FirstList is the list a new household starts with.
const FirstList = "Groceries"

func registerLists(app core.App) {
	// an item's key is what makes it the same thing on a list: "Melk",
	// "melk " and "mélk" are one item, and so are "Melk" and "Lait" when they
	// point at the same generic
	setKey := func(e *core.RecordEvent) error {
		// a name that is a generic's name becomes that generic
		if e.Record.GetString("generic") == "" && e.Record.GetString("product") == "" {
			if g := matchGeneric(e.App, itemKey(e.Record.GetString("name")), "nl",
				listKind(e.App, e.Record.GetString("list"))); g != nil {
				e.Record.Set("generic", g.Id)
			}
		}
		e.Record.Set("key", recordKey(e.App, e.Record))
		return e.Next()
	}
	app.OnRecordCreate("items").BindFunc(setKey)
	// a list is for groceries unless it says otherwise
	app.OnRecordCreate("lists").BindFunc(func(e *core.RecordEvent) error {
		if e.Record.GetString("kind") == "" {
			e.Record.Set("kind", catalogue.Groceries)
		}
		return e.Next()
	})
	app.OnRecordUpdate("items").BindFunc(setKey)

	// ticking an item off records who did it and when, and keeps a purchase
	app.OnRecordUpdateRequest("items").BindFunc(func(e *core.RecordRequestEvent) error {
		was, is := e.Record.Original().GetBool("checked"), e.Record.GetBool("checked")
		if !was && is {
			e.Record.Set("checked_at", tickedAt(e.Record))
			e.Record.Set("checked_by", personID(e.Auth))
		} else if was && !is {
			e.Record.Set("checked_by", "")
		}
		claim(e.Record, personID(e.Auth))
		if err := e.Next(); err != nil {
			return err
		}
		if was && !is {
			// put back on the list from recently bought
			notices.added(e.App, e.Record, personID(e.Auth))
		}
		if !was && is {
			if err := addPurchase(e.App, e.Record, personID(e.Auth)); err != nil {
				e.App.Logger().Error("saving a purchase", "item", e.Record.Id, "error", err)
			}
		}
		return nil
	})

	// a notification still waiting when the server stops is let go
	app.OnTerminate().BindFunc(func(e *core.TerminateEvent) error {
		notices.forget(e.App)
		return e.Next()
	})

	// the first person to sign up gets a list to start with
	app.OnRecordAfterCreateSuccess("users").BindFunc(func(e *core.RecordEvent) error {
		if n, err := e.App.CountRecords("lists"); err == nil && n == 0 {
			lists, err := e.App.FindCollectionByNameOrId("lists")
			if err != nil {
				return err
			}
			l := core.NewRecord(lists)
			l.Set("name", FirstList)
			l.Set("kind", catalogue.Groceries)
			if err := e.App.Save(l); err != nil {
				e.App.Logger().Error("creating the first list", "error", err)
			}
		}
		return e.Next()
	})
}

// claim keeps an item's claim honest: you claim an item for yourself, anyone
// can let a claim go, and bought or put back, it is nobody's.
func claim(item *core.Record, person string) {
	if item.GetBool("checked") || item.Original().GetBool("checked") {
		item.Set("claimed_by", "")
	} else if item.GetString("claimed_by") != item.Original().GetString("claimed_by") &&
		item.GetString("claimed_by") != "" {
		item.Set("claimed_by", person)
	}
}

// offlineTicks is how long a tick can wait on a phone with no signal and still
// keep the time it was made.
const offlineTicks = 7 * 24 * time.Hour

// tickedAt is when an item was ticked off: the time the phone sent, when it
// ticked it off offline and synced later, or now.
func tickedAt(item *core.Record) types.DateTime {
	now := types.NowDateTime()
	sent := item.GetDateTime("checked_at")
	if sent.IsZero() || sent.Equal(item.Original().GetDateTime("checked_at")) {
		return now
	}
	if sent.Time().After(now.Time().Add(time.Minute)) || sent.Time().Before(now.Time().Add(-offlineTicks)) {
		return now
	}
	return sent
}

func registerListRoutes(se *core.ServeEvent) {
	se.Router.POST("/api/app/items", addItem).Bind(apis.RequireAuth())
	se.Router.POST("/api/app/lists/{id}/copy", copyList).Bind(apis.RequireAuth())
}

// newItem is what a phone sends to put something on a list. ID is set when
// the phone made the item while offline, so it keeps its id once it syncs.
type newItem struct {
	ID       string  `json:"id"`
	List     string  `json:"list"`
	Name     string  `json:"name"`
	Generic  string  `json:"generic"`
	Product  string  `json:"product"`
	Quantity *string `json:"quantity"`
	Note     *string `json:"note"`
}

var recordID = regexp.MustCompile(`^[a-z0-9]{15}$`)

// addItem puts something on a list: a generic item, or what was typed, which
// becomes the generic it names when there is one ("lait" is the generic
// milk). If the list already has it, the same item comes back: ticked off
// items return to the list, and a quantity or note sent along replaces the
// old one.
func addItem(e *core.RequestEvent) error {
	var in newItem
	if err := e.BindBody(&in); err != nil {
		return e.BadRequestError("That doesn't look right.", err)
	}
	if in.ID != "" && !recordID.MatchString(in.ID) {
		return e.BadRequestError("That doesn't look right.", nil)
	}
	list, err := e.App.FindRecordById("lists", in.List)
	if err != nil {
		return e.NotFoundError("That list is gone.", err)
	}
	kind := list.GetString("kind")
	if !catalogue.ValidKind(kind) {
		kind = catalogue.Groceries
	}
	lang := personLang(e.Auth)
	name := strings.Join(strings.Fields(in.Name), " ")
	var product, generic *core.Record
	if in.Product != "" {
		p, err := e.App.FindRecordById("products", in.Product)
		if err != nil {
			return e.NotFoundError("That product is gone.", err)
		}
		product = p
	} else if in.Generic != "" {
		g, err := e.App.FindRecordById("generics", in.Generic)
		if err != nil {
			return e.NotFoundError("That item is gone.", err)
		}
		generic = g
	} else {
		generic = matchGeneric(e.App, itemKey(name), lang, kind)
	}
	key := itemKey(name)
	if product != nil {
		key, name = productKey(product), products.Name(product, lang)
	} else if generic != nil {
		key, name = genericKey(generic), genericName(generic, lang)
	}
	if key == "" || name == "" {
		return e.BadRequestError("Type what you need.", nil)
	}

	item, err := e.App.FindFirstRecordByFilter("items", "list = {:list} && key = {:key}",
		dbx.Params{"list": in.List, "key": key})
	status := http.StatusOK
	news := false
	switch {
	case err == nil:
		if item.GetBool("checked") {
			news = true
			item.Set("checked", false)
			item.Set("checked_by", "")
			item.Set("added_by", personID(e.Auth))
		}
	case errors.Is(err, sql.ErrNoRows):
		items, err := e.App.FindCollectionByNameOrId("items")
		if err != nil {
			return err
		}
		item = core.NewRecord(items)
		if in.ID != "" {
			item.Id = in.ID
		}
		item.Set("list", in.List)
		item.Set("name", name)
		if product != nil {
			item.Set("product", product.Id)
		} else if generic != nil {
			item.Set("generic", generic.Id)
		}
		item.Set("added_by", personID(e.Auth))
		status = http.StatusCreated
		news = true
	default:
		return err
	}
	if in.Quantity != nil {
		item.Set("quantity", strings.TrimSpace(*in.Quantity))
	}
	if in.Note != nil {
		item.Set("note", strings.TrimSpace(*in.Note))
	}
	if err := e.App.Save(item); err != nil {
		return e.BadRequestError("That item can't be saved.", err)
	}
	if news {
		notices.added(e.App, item, personID(e.Auth))
	}
	if product != nil {
		// a pinned product picked again comes first next time
		productPicked(e.App, product.Id)
	}
	return e.JSON(status, item)
}

// itemKey is what an item without a generic is found by on its list.
func itemKey(name string) string {
	return food.Normalize(name)
}

// genericKey is what an item of a generic is found by on its list.
func genericKey(generic *core.Record) string {
	return "g:" + generic.GetString("slug")
}

// productKey is what an item of an exact product is found by on its list.
func productKey(product *core.Record) string {
	return "p:" + product.Id
}

// recordKey is an item record's key, from its product, its generic or its
// name.
func recordKey(app core.App, item *core.Record) string {
	if id := item.GetString("product"); id != "" {
		return "p:" + id
	}
	if id := item.GetString("generic"); id != "" {
		if g, err := app.FindRecordById("generics", id); err == nil {
			return genericKey(g)
		}
	}
	return itemKey(item.GetString("name"))
}

// personID is the signed-in person's id, or "" for a worker.
func personID(auth *core.Record) string {
	if auth == nil || auth.Collection().Name != "users" {
		return ""
	}
	return auth.Id
}

func addPurchase(app core.App, item *core.Record, by string) error {
	purchases, err := app.FindCollectionByNameOrId("purchases")
	if err != nil {
		return err
	}
	p := core.NewRecord(purchases)
	p.Set("list", item.GetString("list"))
	p.Set("item", item.Id)
	p.Set("generic", item.GetString("generic"))
	p.Set("product", item.GetString("product"))
	p.Set("name", item.GetString("name"))
	p.Set("key", item.GetString("key"))
	p.Set("price", item.GetFloat("price"))
	p.Set("by", by)
	return app.Save(p)
}
