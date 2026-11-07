package migrations

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

// Kinds of list: groceries, furniture & home, camping, and trips. Each kind
// has its own sections and catalogue items (backend/catalogue), so a sofa on
// a furniture list lands in Furniture, not in Other.
//
// Items, sections and lists show line icons (web/src/lib/icons.ts), named
// here; emojis people picked before still show. The household can pick
// another icon for a section, kept in section_icons by kind and section.
func init() {
	m.Register(func(app core.App) error {
		icon := func(name string) *core.TextField {
			return &core.TextField{Name: name, Max: 40, Pattern: `^[a-z0-9-]*$`}
		}

		lists, err := app.FindCollectionByNameOrId("lists")
		if err != nil {
			return err
		}
		lists.Fields.Add(&core.SelectField{Name: "kind", Values: []string{"groceries", "home", "camping", "trip"}, MaxSelect: 1})
		if err := app.Save(lists); err != nil {
			return err
		}

		generics, err := app.FindCollectionByNameOrId("generics")
		if err != nil {
			return err
		}
		generics.Fields.Add(&core.TextField{Name: "kind", Max: 20}, icon("icon"))
		generics.AddIndex("idx_generics_kind", false, "kind", "")
		if err := app.Save(generics); err != nil {
			return err
		}

		choices, err := app.FindCollectionByNameOrId("item_choices")
		if err != nil {
			return err
		}
		choices.Fields.Add(icon("icon"))
		if err := app.Save(choices); err != nil {
			return err
		}

		// what was there is groceries
		for _, table := range []string{"lists", "generics"} {
			if _, err := app.DB().Update(table, dbx.Params{"kind": "groceries"}, dbx.HashExp{"kind": ""}).Execute(); err != nil {
				return err
			}
		}

		signedIn := types.Pointer(`@request.auth.id != ""`)
		sectionIcons := core.NewBaseCollection("section_icons")
		sectionIcons.Fields.Add(
			// the section's kind of list and key: "home:home-textiles", "groceries:other"
			&core.TextField{Name: "key", Required: true, Max: 80},
			icon("icon"),
			&core.AutodateField{Name: "created", OnCreate: true},
			&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
		)
		sectionIcons.AddIndex("idx_section_icons_key", true, "key", "")
		sectionIcons.ListRule, sectionIcons.ViewRule = signedIn, signedIn
		sectionIcons.CreateRule, sectionIcons.UpdateRule, sectionIcons.DeleteRule = signedIn, signedIn, signedIn
		return app.Save(sectionIcons)
	}, func(app core.App) error {
		if c, err := app.FindCollectionByNameOrId("section_icons"); err == nil {
			if err := app.Delete(c); err != nil {
				return err
			}
		}
		for name, fields := range map[string][]string{
			"lists":        {"kind"},
			"generics":     {"kind", "icon"},
			"item_choices": {"icon"},
		} {
			c, err := app.FindCollectionByNameOrId(name)
			if err != nil {
				return err
			}
			for _, f := range fields {
				c.Fields.RemoveByName(f)
			}
			c.RemoveIndex("idx_generics_kind")
			if err := app.Save(c); err != nil {
				return err
			}
		}
		return nil
	})
}
