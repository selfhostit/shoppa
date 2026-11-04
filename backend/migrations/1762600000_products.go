package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

// Exact products from Open Food Facts (and its sister sites for things that
// aren't food): the ones sold in Belgium, the Netherlands and Greece come
// from the monthly export, others are looked up by barcode and kept. Only the
// server writes them. A product's photo is downloaded the first time a phone
// shows it and kept in thumb, so phones never talk to Open Food Facts.
//
// An item can be an exact product, so the other person knows which one to buy.
func init() {
	m.Register(func(app core.App) error {
		signedIn := types.Pointer(`@request.auth.id != ""`)

		products := core.NewBaseCollection("products")
		products.Fields.Add(
			&core.TextField{Name: "barcode", Required: true, Max: 14, Pattern: `^\d{6,14}$`},
			&core.TextField{Name: "name", Required: true, Max: 200},
			&core.TextField{Name: "name_nl", Max: 200},
			&core.TextField{Name: "name_fr", Max: 200},
			&core.TextField{Name: "brand", Max: 200},
			&core.TextField{Name: "quantity", Max: 50},
			&core.TextField{Name: "categories", Max: 2000},
			&core.TextField{Name: "image_url", Max: 500},
			&core.FileField{Name: "thumb", MaxSelect: 1, MaxSize: 1 << 20,
				MimeTypes: []string{"image/jpeg", "image/png", "image/webp"}},
			// the names, brand and category words, normalized, for search
			&core.TextField{Name: "search", Max: 3000, Hidden: true},
			// off, opf (Open Products Facts) or obf (Open Beauty Facts)
			&core.TextField{Name: "source", Max: 10},
			&core.AutodateField{Name: "created", OnCreate: true},
			&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
		)
		products.AddIndex("idx_products_barcode", true, "barcode", "")
		products.ListRule, products.ViewRule = signedIn, signedIn
		if err := app.Save(products); err != nil {
			return err
		}

		imports := core.NewBaseCollection("product_imports")
		imports.Fields.Add(
			&core.TextField{Name: "name", Required: true, Max: 50},
			&core.TextField{Name: "version", Max: 50},
			&core.NumberField{Name: "products"},
			&core.DateField{Name: "imported"},
			&core.BoolField{Name: "running"},
			&core.TextField{Name: "error", Max: 500},
		)
		imports.AddIndex("idx_product_imports_name", true, "name", "")
		imports.ListRule, imports.ViewRule = signedIn, signedIn
		if err := app.Save(imports); err != nil {
			return err
		}

		for _, name := range []string{"items", "purchases"} {
			c, err := app.FindCollectionByNameOrId(name)
			if err != nil {
				return err
			}
			c.Fields.Add(&core.RelationField{Name: "product", CollectionId: products.Id, MaxSelect: 1})
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
			c.Fields.RemoveByName("product")
			if err := app.Save(c); err != nil {
				return err
			}
		}
		for _, name := range []string{"product_imports", "products"} {
			c, err := app.FindCollectionByNameOrId(name)
			if err != nil {
				return err
			}
			if err := app.Delete(c); err != nil {
				return err
			}
		}
		return nil
	})
}
