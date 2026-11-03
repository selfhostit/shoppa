package app

import (
	"database/sql"
	"errors"
	"net/http"
	"regexp"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
	"github.com/selfhostit/food"
)

// FirstList is the list a new household starts with.
const FirstList = "Groceries"

func registerLists(app core.App) {
	// an item's key is what makes it the same thing on a list: "Melk",
	// "melk " and "mélk" are one item
	setKey := func(e *core.RecordEvent) error {
		e.Record.Set("key", itemKey(e.Record.GetString("name")))
		return e.Next()
	}
	app.OnRecordCreate("items").BindFunc(setKey)
	app.OnRecordUpdate("items").BindFunc(setKey)

	// ticking an item off records who did it and when, and keeps a purchase
	app.OnRecordUpdateRequest("items").BindFunc(func(e *core.RecordRequestEvent) error {
		was, is := e.Record.Original().GetBool("checked"), e.Record.GetBool("checked")
		if !was && is {
			e.Record.Set("checked_at", types.NowDateTime())
			e.Record.Set("checked_by", personID(e.Auth))
		} else if was && !is {
			e.Record.Set("checked_by", "")
		}
		if err := e.Next(); err != nil {
			return err
		}
		if !was && is {
			if err := addPurchase(e.App, e.Record, personID(e.Auth)); err != nil {
				e.App.Logger().Error("saving a purchase", "item", e.Record.Id, "error", err)
			}
		}
		return nil
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
			l.Set("emoji", "🛒")
			if err := e.App.Save(l); err != nil {
				e.App.Logger().Error("creating the first list", "error", err)
			}
		}
		return e.Next()
	})
}

func registerListRoutes(se *core.ServeEvent) {
	se.Router.POST("/api/app/items", addItem).Bind(apis.RequireAuth())
}

// newItem is what a phone sends to put something on a list. ID is set when
// the phone made the item while offline, so it keeps its id once it syncs.
type newItem struct {
	ID       string  `json:"id"`
	List     string  `json:"list"`
	Name     string  `json:"name"`
	Quantity *string `json:"quantity"`
	Note     *string `json:"note"`
}

var recordID = regexp.MustCompile(`^[a-z0-9]{15}$`)

// addItem puts something on a list. If the list already has it, the same
// item comes back: ticked off items return to the list, and a quantity or
// note sent along replaces the old one.
func addItem(e *core.RequestEvent) error {
	var in newItem
	if err := e.BindBody(&in); err != nil {
		return e.BadRequestError("That doesn't look right.", err)
	}
	name := strings.Join(strings.Fields(in.Name), " ")
	key := itemKey(name)
	if key == "" {
		return e.BadRequestError("Type what you need.", nil)
	}
	if in.ID != "" && !recordID.MatchString(in.ID) {
		return e.BadRequestError("That doesn't look right.", nil)
	}
	if _, err := e.App.FindRecordById("lists", in.List); err != nil {
		return e.NotFoundError("That list is gone.", err)
	}

	item, err := e.App.FindFirstRecordByFilter("items", "list = {:list} && key = {:key}",
		dbx.Params{"list": in.List, "key": key})
	status := http.StatusOK
	switch {
	case err == nil:
		if item.GetBool("checked") {
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
		item.Set("added_by", personID(e.Auth))
		status = http.StatusCreated
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
	return e.JSON(status, item)
}

// itemKey is what an item is found by on its list.
func itemKey(name string) string {
	return food.Normalize(name)
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
	p.Set("name", item.GetString("name"))
	p.Set("key", item.GetString("key"))
	p.Set("by", by)
	return app.Save(p)
}
