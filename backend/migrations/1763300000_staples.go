package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

// Staples: items that come back on their list by themselves, every so many
// days ("toilet paper, every 2 weeks"). A staple belongs to the item, so
// taking the item off the list stops it too.
func init() {
	m.Register(func(app core.App) error {
		items, err := app.FindCollectionByNameOrId("items")
		if err != nil {
			return err
		}
		signedIn := types.Pointer(`@request.auth.id != ""`)
		staples := core.NewBaseCollection("staples")
		staples.Fields.Add(
			&core.RelationField{Name: "item", CollectionId: items.Id, MaxSelect: 1, Required: true, CascadeDelete: true},
			&core.NumberField{Name: "every", Required: true, OnlyInt: true, Min: types.Pointer(1.0), Max: types.Pointer(365.0)},
			// when it next comes back; set by the server
			&core.DateField{Name: "next"},
			&core.AutodateField{Name: "created", OnCreate: true},
			&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
		)
		staples.AddIndex("idx_staples_item", true, "item", "")
		staples.AddIndex("idx_staples_next", false, "next", "")
		staples.ListRule, staples.ViewRule = signedIn, signedIn
		staples.CreateRule, staples.UpdateRule, staples.DeleteRule = signedIn, signedIn, signedIn
		return app.Save(staples)
	}, func(app core.App) error {
		staples, err := app.FindCollectionByNameOrId("staples")
		if err != nil {
			return err
		}
		return app.Delete(staples)
	})
}
