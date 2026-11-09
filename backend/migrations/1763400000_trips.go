package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

// Trips: "Anna is shopping at Colruyt". Starting one tells everyone else to
// add what they need now; it ends when the shopper says so, or by itself
// after a few hours.
func init() {
	m.Register(func(app core.App) error {
		lists, err := app.FindCollectionByNameOrId("lists")
		if err != nil {
			return err
		}
		users, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}
		signedIn := types.Pointer(`@request.auth.id != ""`)
		mine := types.Pointer(`by = @request.auth.id`)
		trips := core.NewBaseCollection("trips")
		trips.Fields.Add(
			&core.RelationField{Name: "list", CollectionId: lists.Id, MaxSelect: 1, Required: true, CascadeDelete: true},
			// who is shopping; set by the server
			&core.RelationField{Name: "by", CollectionId: users.Id, MaxSelect: 1, CascadeDelete: true},
			&core.TextField{Name: "store", Max: 40},
			&core.DateField{Name: "ended"},
			&core.AutodateField{Name: "created", OnCreate: true},
			&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
		)
		trips.AddIndex("idx_trips_created", false, "created", "")
		trips.ListRule, trips.ViewRule, trips.CreateRule = signedIn, signedIn, signedIn
		trips.UpdateRule, trips.DeleteRule = mine, mine
		return app.Save(trips)
	}, func(app core.App) error {
		trips, err := app.FindCollectionByNameOrId("trips")
		if err != nil {
			return err
		}
		return app.Delete(trips)
	})
}
