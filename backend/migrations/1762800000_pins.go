package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

// Pins: the household's choice of product for a generic item, "melk" is
// "Boni halfvolle melk", so the next "melk" suggests that product first. A
// generic can have more than one; the one bought most comes first.
func init() {
	m.Register(func(app core.App) error {
		generics, err := app.FindCollectionByNameOrId("generics")
		if err != nil {
			return err
		}
		products, err := app.FindCollectionByNameOrId("products")
		if err != nil {
			return err
		}
		signedIn := types.Pointer(`@request.auth.id != ""`)
		pins := core.NewBaseCollection("pins")
		pins.Fields.Add(
			&core.RelationField{Name: "generic", CollectionId: generics.Id, MaxSelect: 1, Required: true, CascadeDelete: true},
			&core.RelationField{Name: "product", CollectionId: products.Id, MaxSelect: 1, Required: true, CascadeDelete: true},
			&core.NumberField{Name: "uses", OnlyInt: true},
			&core.DateField{Name: "last_used"},
			&core.AutodateField{Name: "created", OnCreate: true},
		)
		pins.AddIndex("idx_pins_generic_product", true, "generic, product", "")
		// pinned through /api/app/items/{id}/pin, which counts the uses;
		// anyone can unpin
		pins.ListRule, pins.ViewRule, pins.DeleteRule = signedIn, signedIn, signedIn
		return app.Save(pins)
	}, func(app core.App) error {
		pins, err := app.FindCollectionByNameOrId("pins")
		if err != nil {
			return err
		}
		return app.Delete(pins)
	})
}
