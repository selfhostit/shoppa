package app

import (
	"net/http"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"github.com/selfhostit/shoppa/catalogue"
)

// copyItemFields are what a copied item keeps: what it is and how much, not
// who added or ticked it.
var copyItemFields = []string{"name", "generic", "product", "quantity", "note"}

// copyList starts a new list from an old one: "the camping trip, again". The
// new list has the same kind and every item of the old one, ticked or not,
// all still to buy. Nobody is notified: these aren't new things to get yet.
func copyList(e *core.RequestEvent) error {
	var in struct {
		Name string `json:"name"`
	}
	if err := e.BindBody(&in); err != nil {
		return e.BadRequestError("That doesn't look right.", err)
	}
	name := strings.Join(strings.Fields(in.Name), " ")
	if name == "" {
		return e.BadRequestError("Give the list a name.", nil)
	}
	from, err := e.App.FindRecordById("lists", e.Request.PathValue("id"))
	if err != nil {
		return e.NotFoundError("That list is gone.", err)
	}
	kind := from.GetString("kind")
	if !catalogue.ValidKind(kind) {
		kind = catalogue.Groceries
	}
	var list *core.Record
	err = e.App.RunInTransaction(func(tx core.App) error {
		lists, err := tx.FindCollectionByNameOrId("lists")
		if err != nil {
			return err
		}
		var last struct {
			Sort float64 `db:"sort"`
		}
		_ = tx.DB().NewQuery("SELECT COALESCE(MAX(sort), 0) AS sort FROM lists").One(&last)
		list = core.NewRecord(lists)
		list.Set("name", name)
		list.Set("kind", kind)
		list.Set("emoji", from.GetString("emoji"))
		list.Set("sort", last.Sort+1)
		if err := tx.Save(list); err != nil {
			return err
		}
		old, err := tx.FindAllRecords("items", dbx.HashExp{"list": from.Id})
		if err != nil {
			return err
		}
		for _, o := range old {
			item := core.NewRecord(o.Collection())
			item.Set("list", list.Id)
			for _, f := range copyItemFields {
				item.Set(f, o.Get(f))
			}
			item.Set("added_by", personID(e.Auth))
			if err := tx.Save(item); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return e.BadRequestError("That list can't be copied.", err)
	}
	return e.JSON(http.StatusCreated, list)
}
