package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

// Prices: what was paid for an item, in euro, kept on its purchase too, so
// the Me tab can add up what each list cost per month.
func init() {
	m.Register(func(app core.App) error {
		for _, name := range []string{"items", "purchases"} {
			c, err := app.FindCollectionByNameOrId(name)
			if err != nil {
				return err
			}
			c.Fields.Add(&core.NumberField{Name: "price", Min: types.Pointer(0.0), Max: types.Pointer(100000.0)})
			if err := app.Save(c); err != nil {
				return err
			}
		}
		return nil
	}, func(app core.App) error {
		for _, name := range []string{"items", "purchases"} {
			c, err := app.FindCollectionByNameOrId(name)
			if err != nil {
				return err
			}
			c.Fields.RemoveByName("price")
			if err := app.Save(c); err != nil {
				return err
			}
		}
		return nil
	})
}
