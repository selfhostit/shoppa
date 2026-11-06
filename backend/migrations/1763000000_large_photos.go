package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// A larger photo of each product, kept the first time someone taps a product's
// picture to see it better. The thumb stays what lists show.
func init() {
	m.Register(func(app core.App) error {
		products, err := app.FindCollectionByNameOrId("products")
		if err != nil {
			return err
		}
		products.Fields.Add(&core.FileField{Name: "photo", MaxSelect: 1, MaxSize: 1 << 20,
			MimeTypes: []string{"image/jpeg", "image/png", "image/webp"}})
		return app.Save(products)
	}, func(app core.App) error {
		products, err := app.FindCollectionByNameOrId("products")
		if err != nil {
			return err
		}
		products.Fields.RemoveByName("photo")
		return app.Save(products)
	})
}
