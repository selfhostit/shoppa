package app_test

import (
	"net/http"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

// newProduct keeps a product as if it came from the export.
func newProduct(t *testing.T, a core.App, barcode, name, nameFR string) *core.Record {
	col, err := a.FindCollectionByNameOrId("products")
	if err != nil {
		t.Fatal(err)
	}
	p := core.NewRecord(col)
	p.Set("barcode", barcode)
	p.Set("name", name)
	p.Set("name_fr", nameFR)
	p.Set("brand", "Boni")
	p.Set("search", "halfvolle melk | lait demi ecreme | boni")
	p.Set("source", "off")
	if err := a.Save(p); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestProductItems(t *testing.T) {
	a := newApp(t)
	defer a.Cleanup()
	ada, adaToken := addUser(t, a, "ada@example.com", "owner")
	bob, bobToken := addUser(t, a, "bob@example.com", "member")
	withLang(t, a, ada, "nl")
	withLang(t, a, bob, "fr")
	list := newList(t, a, "Weekend")
	boni := newProduct(t, a, "5410000000001", "Halfvolle melk", "Lait demi-écrémé")

	// an exact product, named in the adder's language
	item := send(t, a, http.StatusCreated, http.MethodPost, "/api/app/items", bobToken,
		`{"list":"`+list+`","product":"`+boni.Id+`"}`)
	if item["product"] != boni.Id || item["name"] != "Lait demi-écrémé" || item["generic"] != "" {
		t.Fatalf("item %v", item)
	}
	// the same product again is the same item, and isn't the generic milk
	again := send(t, a, http.StatusOK, http.MethodPost, "/api/app/items", adaToken,
		`{"list":"`+list+`","product":"`+boni.Id+`","quantity":"2×"}`)
	if again["id"] != item["id"] || again["quantity"] != "2×" {
		t.Fatalf("again %v", again)
	}
	if milk := send(t, a, http.StatusCreated, http.MethodPost, "/api/app/items", adaToken,
		`{"list":"`+list+`","name":"melk"}`); milk["id"] == item["id"] {
		t.Errorf("the generic milk is the product")
	}
	send(t, a, http.StatusNotFound, http.MethodPost, "/api/app/items", adaToken, `{"list":"`+list+`","product":"nosuchproduct12"}`)

	// buying it keeps the product with the purchase
	send(t, a, http.StatusOK, http.MethodPatch, "/api/collections/items/records/"+item["id"].(string), adaToken, `{"checked":true}`)
	if p, err := a.FindFirstRecordByData("purchases", "item", item["id"]); err != nil || p.GetString("product") != boni.Id {
		t.Errorf("purchase %v, %v", p, err)
	}
}

func TestProductRoutes(t *testing.T) {
	a := newApp(t)
	defer a.Cleanup()
	_, owner := addUser(t, a, "ada@example.com", "owner")
	_, member := addUser(t, a, "bob@example.com", "member")
	boni := newProduct(t, a, "5410000000001", "Halfvolle melk", "Lait demi-écrémé")
	factory := func(testing.TB) *tests.TestApp { return a }

	scenarios := []tests.ApiScenario{
		{
			Name:            "search finds products",
			Method:          http.MethodGet,
			URL:             "/api/app/search?q=melk",
			Headers:         map[string]string{"Authorization": member},
			ExpectedStatus:  http.StatusOK,
			ExpectedContent: []string{`"id":"` + boni.Id + `"`, `"brand":"Boni"`},
		},
		{
			Name:            "search needs a person",
			Method:          http.MethodGet,
			URL:             "/api/app/search?q=melk",
			ExpectedStatus:  http.StatusUnauthorized,
			ExpectedContent: []string{`"status":401`},
		},
		{
			Name:            "a kept barcode",
			Method:          http.MethodGet,
			URL:             "/api/app/barcode/5410000000001",
			Headers:         map[string]string{"Authorization": member},
			ExpectedStatus:  http.StatusOK,
			ExpectedContent: []string{`"id":"` + boni.Id + `"`},
		},
		{
			Name:            "not a barcode",
			Method:          http.MethodGet,
			URL:             "/api/app/barcode/abc",
			Headers:         map[string]string{"Authorization": member},
			ExpectedStatus:  http.StatusNotFound,
			ExpectedContent: []string{`"status":404`},
		},
		{
			Name:            "a product without a photo",
			Method:          http.MethodGet,
			URL:             "/api/app/products/" + boni.Id + "/photo",
			ExpectedStatus:  http.StatusNotFound,
			ExpectedContent: []string{`"status":404`},
		},
		{
			Name:            "a large photo of a product without a photo",
			Method:          http.MethodGet,
			URL:             "/api/app/products/" + boni.Id + "/photo?size=large",
			ExpectedStatus:  http.StatusNotFound,
			ExpectedContent: []string{`"status":404`},
		},
		{
			Name:            "members can't start an import",
			Method:          http.MethodPost,
			URL:             "/api/app/products/import",
			Headers:         map[string]string{"Authorization": member},
			ExpectedStatus:  http.StatusForbidden,
			ExpectedContent: []string{"Only the owner"},
		},
		{
			Name:            "nobody writes products by hand",
			Method:          http.MethodPost,
			URL:             "/api/collections/products/records",
			Headers:         map[string]string{"Authorization": owner},
			Body:            nil,
			ExpectedStatus:  http.StatusForbidden,
			ExpectedContent: []string{`"status":403`},
		},
	}
	for i := range scenarios {
		s := scenarios[i]
		s.TestAppFactory = factory
		s.DisableTestAppCleanup = true
		s.Test(t)
	}
}
