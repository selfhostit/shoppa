package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Who a trip told: the people with notifications on when it started, so the
// shopper sees who got the news, and those people hear when it's done.
func init() {
	m.Register(func(app core.App) error {
		trips, err := app.FindCollectionByNameOrId("trips")
		if err != nil {
			return err
		}
		users, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}
		// set by the server
		trips.Fields.Add(&core.RelationField{Name: "notified", CollectionId: users.Id, MaxSelect: 99})
		return app.Save(trips)
	}, func(app core.App) error {
		trips, err := app.FindCollectionByNameOrId("trips")
		if err != nil {
			return err
		}
		trips.Fields.RemoveByName("notified")
		return app.Save(trips)
	})
}
