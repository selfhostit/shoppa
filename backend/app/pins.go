package app

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"

	"github.com/selfhostit/shoppa/products"
)

func registerPinRoutes(se *core.ServeEvent) {
	se.Router.POST("/api/app/items/{id}/pin", pinItem).Bind(apis.RequireAuth())
}

// pinItem makes a generic item ("Melk") the exact product picked for it
// ("Boni halfvolle melk"), and remembers the choice: the next "melk"
// suggests that product first. When the product is already on the list, the
// two become one.
func pinItem(e *core.RequestEvent) error {
	var in struct {
		Product string `json:"product"`
	}
	if err := e.BindBody(&in); err != nil {
		return e.BadRequestError("That doesn't look right.", err)
	}
	item, err := e.App.FindRecordById("items", e.Request.PathValue("id"))
	if err != nil {
		return e.NotFoundError("That item is gone.", err)
	}
	product, err := e.App.FindRecordById("products", in.Product)
	if err != nil {
		return e.NotFoundError("That product is gone.", err)
	}
	generic := item.GetString("generic")
	if generic == "" {
		return e.BadRequestError("Only a usual item like “milk” can be pinned to a product.", nil)
	}

	var result *core.Record
	err = e.App.RunInTransaction(func(tx core.App) error {
		if err := usePin(tx, generic, product.Id, true); err != nil {
			return err
		}
		existing, err := tx.FindFirstRecordByFilter("items", "list = {:list} && key = {:key}",
			dbx.Params{"list": item.GetString("list"), "key": productKey(product)})
		switch {
		case err == nil:
			// already on the list: that one stays, with this one's quantity
			// and note when it has none
			for _, f := range []string{"quantity", "note"} {
				if existing.GetString(f) == "" {
					existing.Set(f, item.GetString(f))
				}
			}
			if existing.GetBool("checked") {
				existing.Set("checked", false)
				existing.Set("checked_by", "")
			}
			if err := tx.Delete(item); err != nil {
				return err
			}
			result = existing
		case errors.Is(err, sql.ErrNoRows):
			item.Set("product", product.Id)
			item.Set("generic", "")
			item.Set("name", products.Name(product, personLang(e.Auth)))
			result = item
		default:
			return err
		}
		return tx.Save(result)
	})
	if err != nil {
		return e.BadRequestError("That can't be pinned.", err)
	}
	return e.JSON(http.StatusOK, result)
}

// usePin counts a use of a product picked for a generic. With create, it pins
// the product when it isn't yet; without, only an existing pin counts.
func usePin(app core.App, generic, product string, create bool) error {
	pin, err := app.FindFirstRecordByFilter("pins", "generic = {:g} && product = {:p}",
		dbx.Params{"g": generic, "p": product})
	if errors.Is(err, sql.ErrNoRows) {
		if !create {
			return nil
		}
		pins, err := app.FindCollectionByNameOrId("pins")
		if err != nil {
			return err
		}
		pin = core.NewRecord(pins)
		pin.Set("generic", generic)
		pin.Set("product", product)
	} else if err != nil {
		return err
	}
	pin.Set("uses", pin.GetInt("uses")+1)
	pin.Set("last_used", types.NowDateTime())
	return app.Save(pin)
}

// productPicked counts a use of every pin of a product that went on a list.
func productPicked(app core.App, product string) {
	var pins []*core.Record
	if err := app.RecordQuery("pins").AndWhere(dbx.HashExp{"product": product}).All(&pins); err != nil {
		return
	}
	for _, p := range pins {
		if err := usePin(app, p.GetString("generic"), product, false); err != nil {
			app.Logger().Error("counting a pin", "pin", p.Id, "error", err)
		}
	}
}
