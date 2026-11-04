package app

import (
	"errors"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"

	"github.com/selfhostit/shoppa/jobs"
	"github.com/selfhostit/shoppa/products"
)

const productsJob = "products"

// AutoImport starts the first product import a minute after a new install
// starts. Tests turn it off, and so does SHOPPA_AUTO_IMPORT=off.
var AutoImport = os.Getenv("SHOPPA_AUTO_IMPORT") != "off"

// Products from Open Food Facts: the monthly import, search, barcode lookups
// and photos kept at home.
func registerProducts(app core.App) {
	products.UserAgent = "Shoppa/" + Version + " (self-hosted shopping list)"

	// on the 10th of every month, at night: a week after Troye's, so a NAS
	// running both never reads the export twice at once
	jobs.MustAdd(app, productsJob, "30 4 10 * *", products.Import)

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		// an import the server stopped in the middle of isn't running anymore
		if _, err := se.App.DB().Update("product_imports", dbx.Params{"running": false}, dbx.HashExp{"running": true}).Execute(); err != nil {
			se.App.Logger().Warn("Could not reset the product import", "error", err)
		}
		// a new install gets its products without waiting for the 10th
		if n, err := se.App.CountRecords("product_imports"); AutoImport && err == nil && n == 0 {
			time.AfterFunc(time.Minute, func() { jobs.Run(se.App, productsJob, products.Import) })
		}

		users := apis.RequireAuth("users")
		se.Router.GET("/api/app/search", searchProducts).Bind(users)
		se.Router.GET("/api/app/barcode/{code}", lookupBarcode).Bind(users)
		se.Router.POST("/api/app/products/import", startImport).Bind(users)
		// a photo is an <img>, which can't send a token: the route is open,
		// but only shows photos of products already kept
		se.Router.GET("/api/app/products/{id}/photo", productPhoto)
		return se.Next()
	})
}

func searchProducts(e *core.RequestEvent) error {
	limit, err := strconv.Atoi(e.Request.URL.Query().Get("limit"))
	if err != nil || limit <= 0 || limit > 50 {
		limit = 20
	}
	found, err := products.Search(e.App, e.Request.URL.Query().Get("q"), personLang(e.Auth), limit)
	if err != nil {
		return e.InternalServerError("", err)
	}
	return e.JSON(http.StatusOK, found)
}

func lookupBarcode(e *core.RequestEvent) error {
	p, err := products.Lookup(e.App, e.Request.PathValue("code"))
	if errors.Is(err, products.ErrUnknown) {
		return e.NotFoundError("That barcode isn't in Open Food Facts yet.", err)
	}
	if err != nil {
		return e.Error(http.StatusBadGateway, "Open Food Facts can't be reached right now.", err)
	}
	return e.JSON(http.StatusOK, p)
}

func startImport(e *core.RequestEvent) error {
	if e.Auth.GetString("role") != "owner" {
		return e.ForbiddenError("Only the owner can update the products.", nil)
	}
	if status, err := e.App.FindFirstRecordByData("product_imports", "name", products.ImportName); err == nil && status.GetBool("running") {
		return e.BadRequestError("The products are already being updated.", nil)
	}
	go jobs.Run(e.App, productsJob, products.Import)
	return e.NoContent(http.StatusAccepted)
}

// productPhoto sends the product's photo, downloading it from Open Food Facts
// the first time and keeping it.
func productPhoto(e *core.RequestEvent) error {
	p, err := e.App.FindRecordById("products", e.Request.PathValue("id"))
	if err != nil {
		return e.NotFoundError("", err)
	}
	if err := products.KeepPhoto(e.App, p); err != nil {
		return e.NotFoundError("", err)
	}
	e.Response.Header().Set("Cache-Control", "public, max-age=2592000")
	return e.Redirect(http.StatusFound, "/api/files/"+p.Collection().Id+"/"+p.Id+"/"+p.GetString("thumb"))
}
