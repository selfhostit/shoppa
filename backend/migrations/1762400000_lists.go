package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

// Shopping lists, their items and every purchase. One install is one
// household: everyone signed in sees and changes every list.
//
// A list holds one item per thing, found by its key (the normalized name).
// Ticking an item off keeps it, checked, under "recently bought", and one tap
// there puts it back. Every tick is also kept as a purchase, which version 2
// learns usual items and aisle order from.
func init() {
	m.Register(func(app core.App) error {
		users, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}
		signedIn := types.Pointer(`@request.auth.id != ""`)

		lists := core.NewBaseCollection("lists")
		lists.Fields.Add(
			&core.TextField{Name: "name", Required: true, Max: 60},
			&core.TextField{Name: "emoji", Max: 16},
			&core.NumberField{Name: "sort"},
			&core.BoolField{Name: "archived"},
			&core.AutodateField{Name: "created", OnCreate: true},
			&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
		)
		lists.ListRule, lists.ViewRule, lists.CreateRule, lists.UpdateRule, lists.DeleteRule =
			signedIn, signedIn, signedIn, signedIn, signedIn
		if err := app.Save(lists); err != nil {
			return err
		}

		items := core.NewBaseCollection("items")
		items.Fields.Add(
			&core.RelationField{Name: "list", CollectionId: lists.Id, MaxSelect: 1, Required: true, CascadeDelete: true},
			&core.TextField{Name: "name", Required: true, Max: 120},
			// what makes two items the same thing on a list; set by the server
			&core.TextField{Name: "key", Max: 200, Hidden: true},
			&core.TextField{Name: "quantity", Max: 20},
			&core.TextField{Name: "note", Max: 200},
			&core.BoolField{Name: "checked"},
			&core.DateField{Name: "checked_at"},
			&core.RelationField{Name: "checked_by", CollectionId: users.Id, MaxSelect: 1},
			&core.RelationField{Name: "added_by", CollectionId: users.Id, MaxSelect: 1},
			&core.AutodateField{Name: "created", OnCreate: true},
			&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
		)
		items.AddIndex("idx_items_list_key", true, "list, key", "")
		items.AddIndex("idx_items_list_checked", false, "list, checked, checked_at", "")
		// items are added through /api/app/items, which finds the one already
		// on the list; the rest is plain record updates
		items.ListRule, items.ViewRule, items.UpdateRule, items.DeleteRule = signedIn, signedIn, signedIn, signedIn
		items.CreateRule = nil
		if err := app.Save(items); err != nil {
			return err
		}

		purchases := core.NewBaseCollection("purchases")
		purchases.Fields.Add(
			&core.RelationField{Name: "list", CollectionId: lists.Id, MaxSelect: 1, Required: true, CascadeDelete: true},
			&core.RelationField{Name: "item", CollectionId: items.Id, MaxSelect: 1},
			&core.TextField{Name: "name", Max: 120},
			&core.TextField{Name: "key", Max: 200},
			&core.RelationField{Name: "by", CollectionId: users.Id, MaxSelect: 1},
			&core.AutodateField{Name: "created", OnCreate: true},
		)
		purchases.AddIndex("idx_purchases_key", false, "key, created", "")
		// written by the server when an item is ticked off, never by hand
		purchases.ListRule, purchases.ViewRule = signedIn, signedIn
		return app.Save(purchases)
	}, func(app core.App) error {
		for _, name := range []string{"purchases", "items", "lists"} {
			c, err := app.FindCollectionByNameOrId(name)
			if err != nil {
				return err
			}
			if err := app.Delete(c); err != nil {
				return err
			}
		}
		return nil
	})
}
