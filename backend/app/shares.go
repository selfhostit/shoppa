package app

import (
	"net/http"
	"slices"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/security"
	"github.com/pocketbase/pocketbase/tools/types"
)

// sharedBought is how long a ticked off item still shows on a shared list.
const sharedBought = 24 * time.Hour

var langs = []string{"en", "nl", "fr", "el"}

func registerShares(app core.App) {
	app.OnRecordCreateRequest("shares").BindFunc(func(e *core.RecordRequestEvent) error {
		e.Record.Set("token", security.RandomString(32))
		e.Record.Set("by", personID(e.Auth))
		return e.Next()
	})
}

// The routes a guest's link uses: no account, only the token.
func registerShareRoutes(se *core.ServeEvent) {
	se.Router.GET("/api/app/shared/{token}", sharedList)
	se.Router.POST("/api/app/shared/{token}/items/{id}", sharedTick)
}

// SharedItem is an item as a guest sees it: no one's names, no ids of people.
type SharedItem struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Quantity string `json:"quantity"`
	Note     string `json:"note"`
	Checked  bool   `json:"checked"`
}

// SharedList is a list as a guest sees it.
type SharedList struct {
	Name    string       `json:"name"`
	Kind    string       `json:"kind"`
	CanTick bool         `json:"can_tick"`
	Items   []SharedItem `json:"items"`
}

// findShare is the share a token opens, or a not found error.
func findShare(e *core.RequestEvent) (*core.Record, error) {
	token := e.Request.PathValue("token")
	if len(token) < 32 {
		return nil, e.NotFoundError("This link doesn't work anymore.", nil)
	}
	share, err := e.App.FindFirstRecordByData("shares", "token", token)
	if err != nil {
		return nil, e.NotFoundError("This link doesn't work anymore.", err)
	}
	return share, nil
}

// sharedList shows a shared list: what is still to buy, and what was bought
// in the last day, with names in the language asked for.
func sharedList(e *core.RequestEvent) error {
	share, err := findShare(e)
	if err != nil {
		return err
	}
	list, err := e.App.FindRecordById("lists", share.GetString("list"))
	if err != nil {
		return e.NotFoundError("This link doesn't work anymore.", err)
	}
	lang := e.Request.URL.Query().Get("lang")
	if !slices.Contains(langs, lang) {
		lang = "en"
	}
	since := types.NowDateTime().Add(-sharedBought).String()
	records, err := e.App.FindRecordsByFilter("items",
		"list = {:list} && (checked = false || checked_at >= {:since})", "created", 0, 0,
		dbx.Params{"list": list.Id, "since": since})
	if err != nil {
		return err
	}
	out := SharedList{
		Name:    list.GetString("name"),
		Kind:    list.GetString("kind"),
		CanTick: share.GetBool("can_tick"),
		Items:   make([]SharedItem, len(records)),
	}
	for i, r := range records {
		out.Items[i] = SharedItem{
			ID:       r.Id,
			Name:     itemName(e.App, r, lang),
			Quantity: r.GetString("quantity"),
			Note:     r.GetString("note"),
			Checked:  r.GetBool("checked"),
		}
	}
	return e.JSON(http.StatusOK, out)
}

// sharedTick ticks an item off, or puts it back, for a guest whose link lets
// them.
func sharedTick(e *core.RequestEvent) error {
	share, err := findShare(e)
	if err != nil {
		return err
	}
	if !share.GetBool("can_tick") {
		return e.ForbiddenError("This link only shows the list.", nil)
	}
	var in struct {
		Checked bool `json:"checked"`
	}
	if err := e.BindBody(&in); err != nil {
		return e.BadRequestError("That doesn't look right.", err)
	}
	item, err := e.App.FindRecordById("items", e.Request.PathValue("id"))
	if err != nil || item.GetString("list") != share.GetString("list") {
		return e.NotFoundError("That item is gone.", err)
	}
	if item.GetBool("checked") == in.Checked {
		return e.NoContent(http.StatusNoContent)
	}
	item.Set("checked", in.Checked)
	item.Set("checked_by", "")
	item.Set("claimed_by", "")
	if in.Checked {
		item.Set("checked_at", types.NowDateTime())
	}
	if err := e.App.Save(item); err != nil {
		return e.BadRequestError("That item can't be saved.", err)
	}
	if in.Checked {
		if err := addPurchase(e.App, item, ""); err != nil {
			e.App.Logger().Error("saving a purchase", "item", item.Id, "error", err)
		}
	}
	return e.NoContent(http.StatusNoContent)
}
