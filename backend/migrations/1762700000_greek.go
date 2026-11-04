package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Greek, next to Dutch, French and English: a Greek name for every generic
// item, and Greek as a language a person sees item names in.
func init() {
	m.Register(func(app core.App) error {
		generics, err := app.FindCollectionByNameOrId("generics")
		if err != nil {
			return err
		}
		generics.Fields.Add(&core.TextField{Name: "name_el", Max: 120})
		if err := app.Save(generics); err != nil {
			return err
		}
		return setLangs(app, "en", "nl", "fr", "el")
	}, func(app core.App) error {
		if _, err := app.DB().NewQuery("UPDATE {{users}} SET [[lang]] = '' WHERE [[lang]] = 'el'").Execute(); err != nil {
			return err
		}
		if err := setLangs(app, "en", "nl", "fr"); err != nil {
			return err
		}
		generics, err := app.FindCollectionByNameOrId("generics")
		if err != nil {
			return err
		}
		generics.Fields.RemoveByName("name_el")
		return app.Save(generics)
	})
}

func setLangs(app core.App, langs ...string) error {
	users, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		return err
	}
	users.Fields.GetByName("lang").(*core.SelectField).Values = langs
	return app.Save(users)
}
