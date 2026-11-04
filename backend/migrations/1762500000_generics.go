package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

// Generic items: "melk", "bananen", "wc-papier", with a name in each language,
// a store section and an emoji. The app keeps the bundled ones in step with
// backend/catalogue/generics.json; people can add their own (custom), and
// only those can be changed or deleted.
//
// An item can point at a generic, so "Melk", "Lait" and "Milk" are one item,
// shown in each person's language (users.lang).
func init() {
	m.Register(func(app core.App) error {
		signedIn := `@request.auth.id != ""`
		generics := core.NewBaseCollection("generics")
		generics.Fields.Add(
			&core.TextField{Name: "slug", Required: true, Max: 80, Pattern: `^[a-z0-9-]+$`},
			&core.TextField{Name: "section", Max: 30},
			&core.TextField{Name: "emoji", Max: 16},
			&core.TextField{Name: "name_en", Max: 120},
			&core.TextField{Name: "name_nl", Max: 120},
			&core.TextField{Name: "name_fr", Max: 120},
			&core.TextField{Name: "aliases", Max: 500},
			&core.TextField{Name: "off_categories", Max: 1000},
			// every name, normalized, between bars: "|milk|melk|lait|"
			&core.TextField{Name: "names", Max: 2000},
			&core.BoolField{Name: "custom"},
			&core.AutodateField{Name: "created", OnCreate: true},
			&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
		)
		generics.AddIndex("idx_generics_slug", true, "slug", "")
		generics.ListRule = types.Pointer(signedIn)
		generics.ViewRule = types.Pointer(signedIn)
		generics.CreateRule = types.Pointer(signedIn)
		generics.UpdateRule = types.Pointer(signedIn + ` && custom = true`)
		generics.DeleteRule = types.Pointer(signedIn + ` && custom = true`)
		if err := app.Save(generics); err != nil {
			return err
		}

		for _, name := range []string{"items", "purchases"} {
			c, err := app.FindCollectionByNameOrId(name)
			if err != nil {
				return err
			}
			c.Fields.Add(&core.RelationField{Name: "generic", CollectionId: generics.Id, MaxSelect: 1})
			if err := app.Save(c); err != nil {
				return err
			}
		}

		users, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}
		users.Fields.Add(&core.SelectField{Name: "lang", Values: []string{"en", "nl", "fr"}, MaxSelect: 1})
		return app.Save(users)
	}, func(app core.App) error {
		users, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}
		users.Fields.RemoveByName("lang")
		if err := app.Save(users); err != nil {
			return err
		}
		for _, name := range []string{"items", "purchases"} {
			c, err := app.FindCollectionByNameOrId(name)
			if err != nil {
				return err
			}
			c.Fields.RemoveByName("generic")
			if err := app.Save(c); err != nil {
				return err
			}
		}
		generics, err := app.FindCollectionByNameOrId("generics")
		if err != nil {
			return err
		}
		return app.Delete(generics)
	})
}
