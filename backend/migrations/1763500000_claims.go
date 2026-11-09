package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Claims: "I'll get this one at the bakery". An item someone claimed shows
// whose it is, so two people don't buy the same thing.
func init() {
	m.Register(func(app core.App) error {
		users, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}
		items, err := app.FindCollectionByNameOrId("items")
		if err != nil {
			return err
		}
		items.Fields.Add(&core.RelationField{Name: "claimed_by", CollectionId: users.Id, MaxSelect: 1})
		return app.Save(items)
	}, func(app core.App) error {
		items, err := app.FindCollectionByNameOrId("items")
		if err != nil {
			return err
		}
		items.Fields.RemoveByName("claimed_by")
		return app.Save(items)
	})
}
