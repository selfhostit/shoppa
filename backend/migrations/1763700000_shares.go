package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

// Shares: a link that shows one list to someone outside the household, a
// babysitter or a grandparent, with no account. The link holds a long random
// token; a share can also let them tick things off. Deleting the share stops
// the link.
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
		shares := core.NewBaseCollection("shares")
		shares.Fields.Add(
			&core.RelationField{Name: "list", CollectionId: lists.Id, MaxSelect: 1, Required: true, CascadeDelete: true},
			// set by the server
			&core.TextField{Name: "token", Max: 64},
			&core.BoolField{Name: "can_tick"},
			&core.RelationField{Name: "by", CollectionId: users.Id, MaxSelect: 1},
			&core.AutodateField{Name: "created", OnCreate: true},
		)
		shares.AddIndex("idx_shares_token", true, "token", "")
		shares.ListRule, shares.ViewRule, shares.CreateRule, shares.DeleteRule =
			signedIn, signedIn, signedIn, signedIn
		return app.Save(shares)
	}, func(app core.App) error {
		shares, err := app.FindCollectionByNameOrId("shares")
		if err != nil {
			return err
		}
		return app.Delete(shares)
	})
}
