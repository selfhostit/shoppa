package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

// Item choices: the store section and icon the household picked for an item,
// by its name, so "chicken breast small" stays in Meat & fish with the icon
// they chose every time it is added, whatever it would get on its own.
func init() {
	m.Register(func(app core.App) error {
		signedIn := types.Pointer(`@request.auth.id != ""`)
		choices := core.NewBaseCollection("item_choices")
		choices.Fields.Add(
			// the item's name, normalized: lower case, without accents or punctuation
			&core.TextField{Name: "key", Required: true, Max: 120},
			// empty when only the other one was picked
			&core.TextField{Name: "section", Max: 40},
			&core.TextField{Name: "emoji", Max: 16},
			&core.AutodateField{Name: "created", OnCreate: true},
			&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
		)
		choices.AddIndex("idx_item_choices_key", true, "key", "")
		choices.ListRule, choices.ViewRule = signedIn, signedIn
		choices.CreateRule, choices.UpdateRule, choices.DeleteRule = signedIn, signedIn, signedIn
		return app.Save(choices)
	}, func(app core.App) error {
		choices, err := app.FindCollectionByNameOrId("item_choices")
		if err != nil {
			return err
		}
		return app.Delete(choices)
	})
}
