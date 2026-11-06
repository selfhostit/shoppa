package products_test

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"

	"github.com/selfhostit/shoppa/catalogue"
	_ "github.com/selfhostit/shoppa/migrations"
	"github.com/selfhostit/shoppa/products"
)

func newApp(t *testing.T) *tests.TestApp {
	a, err := tests.NewTestAppWithConfig(core.BaseAppConfig{DataDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(a.Cleanup)
	if err := catalogue.Sync(a); err != nil {
		t.Fatal(err)
	}
	return a
}

// the export's columns that matter, and a product per country
var export = strings.Join([]string{
	"code\tproduct_name\tproduct_name_nl\tproduct_name_fr\tbrands\tquantity\tcategories_tags\tcountries_tags\timage_small_url\tenergy-kcal_100g",
	"5410000000001\tHalfvolle melk\tHalfvolle melk\tLait demi-écrémé\tBoni\t1 L\ten:dairies,en:milks,en:semi-skimmed-milks\ten:belgium\thttps://images.example/541.jpg\t46",
	"8710000000001\tJonge Gouda kaas\t\t\tAH\t500 g\ten:dairies,en:cheeses,en:goudas\ten:netherlands\t\t356",
	"5200000000001\tΓάλα πλήρες\t\t\tΔέλτα\t1 l\ten:milks\ten:greece\t\t",
	"5410000000002\tAfwasmiddel citroen\t\t\tDreft\t500 ml\ten:dishwashing-liquids\ten:belgium\t\t",
	"3000000000001\tCroissant\t\t\t\t\ten:croissants\ten:france\t\t400",
}, "\n")

func serveExport(t *testing.T) {
	var gz bytes.Buffer
	zw := gzip.NewWriter(&gz)
	zw.Write([]byte(export))
	zw.Close()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.UserAgent(), "Shoppa/") {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		w.Write(gz.Bytes())
	}))
	t.Cleanup(srv.Close)
	old := products.ExportURL
	products.ExportURL = srv.URL + "/products.csv.gz"
	t.Cleanup(func() { products.ExportURL = old })
}

func TestImport(t *testing.T) {
	a := newApp(t)
	serveExport(t)
	if err := products.Import(a); err != nil {
		t.Fatal(err)
	}
	// Belgium, the Netherlands and Greece, not France
	if n, _ := a.CountRecords("products"); n != 4 {
		t.Fatalf("%d products", n)
	}
	melk, err := a.FindFirstRecordByData("products", "barcode", "5410000000001")
	if err != nil {
		t.Fatal(err)
	}
	if melk.GetString("name") != "Halfvolle melk" || melk.GetString("name_fr") != "Lait demi-écrémé" ||
		melk.GetString("brand") != "Boni" || melk.GetString("quantity") != "1 L" ||
		melk.GetString("image_url") != "https://images.example/541.jpg" || melk.GetString("source") != "off" {
		t.Errorf("melk %v", melk.PublicExport())
	}
	status, err := a.FindFirstRecordByData("product_imports", "name", products.ImportName)
	if err != nil || status.GetInt("products") != 4 || status.GetBool("running") || status.GetString("error") != "" {
		t.Errorf("status %v, %v", status, err)
	}

	// importing again updates the products and keeps their ids and photos
	melk.Set("thumb", "kept.jpg")
	if err := a.UnsafeWithoutHooks().SaveNoValidate(melk); err != nil {
		t.Fatal(err)
	}
	if err := products.Import(a); err != nil {
		t.Fatal(err)
	}
	again, _ := a.FindFirstRecordByData("products", "barcode", "5410000000001")
	if n, _ := a.CountRecords("products"); n != 4 || again.Id != melk.Id || again.GetString("thumb") != "kept.jpg" {
		t.Errorf("%d products, %v", n, again.PublicExport())
	}
}

func TestSearch(t *testing.T) {
	a := newApp(t)
	serveExport(t)
	if err := products.Import(a); err != nil {
		t.Fatal(err)
	}
	first := func(q, lang string) string {
		t.Helper()
		found, err := products.Search(a, q, lang, 5)
		if err != nil {
			t.Fatal(err)
		}
		if len(found) == 0 {
			return ""
		}
		return found[0].GetString("barcode")
	}
	for _, c := range []struct{ q, lang, want string }{
		{"melk", "nl", "5410000000001"}, // its name
		{"boni", "nl", "5410000000001"}, // its brand
		{"milk", "en", "5410000000001"}, // its English category words
		{"semi skimmed", "en", "5410000000001"},
		{"lait", "fr", "5410000000001"},   // its French name
		{"cheese", "en", "8710000000001"}, // the catalogue's cheese is "kaas" in Dutch
		{"kaas", "nl", "8710000000001"},
		{"γαλα", "en", "5200000000001"},       // Greek, without the accent
		{"washing up", "en", "5410000000002"}, // the catalogue's washing-up liquid is "afwasmiddel"
	} {
		if got := first(c.q, c.lang); got != c.want {
			t.Errorf("%q: %q, want %q", c.q, got, c.want)
		}
	}
	if got := first("croissant", "fr"); got != "" {
		t.Errorf("a French-only product was kept: %s", got)
	}
	if found, _ := products.Search(a, "  ", "en", 5); len(found) != 0 {
		t.Errorf("nothing typed: %d", len(found))
	}
}

func TestLookup(t *testing.T) {
	a := newApp(t)
	var asked []string
	site := func(known map[string]string) string {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			asked = append(asked, r.Host+r.URL.Path)
			for code, body := range known {
				if strings.Contains(r.URL.Path, "/"+code+".json") {
					w.Write([]byte(body))
					return
				}
			}
			w.WriteHeader(http.StatusNotFound)
		}))
		t.Cleanup(srv.Close)
		return srv.URL
	}
	old := products.Sites
	products.Sites = []products.Site{
		{Source: "off", URL: site(map[string]string{
			"5410000000009": `{"status":1,"product":{"product_name":"Speculoos","brands":"Lotus","quantity":"250 g"}}`,
		})},
		{Source: "opf", URL: site(map[string]string{
			"5410000000008": `{"status":1,"product":{"product_name":"Wc-papier","quantity":"8 rollen"}}`,
		})},
	}
	t.Cleanup(func() { products.Sites = old })

	speculoos, err := products.Lookup(a, "5410000000009")
	if err != nil || speculoos.GetString("name") != "Speculoos" || speculoos.GetString("source") != "off" {
		t.Fatalf("speculoos %v, %v", speculoos, err)
	}
	// not food: Open Products Facts has it
	paper, err := products.Lookup(a, "5410000000008")
	if err != nil || paper.GetString("source") != "opf" || paper.GetString("quantity") != "8 rollen" {
		t.Fatalf("paper %v, %v", paper, err)
	}
	// kept: no second lookup, and a UPC-A code finds its EAN-13
	asked = nil
	if again, err := products.Lookup(a, "5410000000009"); err != nil || again.Id != speculoos.Id || len(asked) != 0 {
		t.Errorf("again %v, %v, asked %v", again, err, asked)
	}
	if _, err := products.Lookup(a, "1234567890123"); !errors.Is(err, products.ErrUnknown) {
		t.Errorf("unknown: %v", err)
	}
	if _, err := products.Lookup(a, "../etc"); !errors.Is(err, products.ErrUnknown) {
		t.Errorf("not a barcode: %v", err)
	}
}

// a 1×1 PNG
var png, _ = base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==")

func TestKeepPhoto(t *testing.T) {
	a := newApp(t)
	downloads := 0
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		downloads++
		if r.URL.Path == "/text" {
			w.Write([]byte("not a photo"))
			return
		}
		w.Write(png)
	}))
	defer srv.Close()
	products.SetPhotoClient(srv.Client())

	col, _ := a.FindCollectionByNameOrId("products")
	p := core.NewRecord(col)
	p.Set("barcode", "5410000000001")
	p.Set("name", "Halfvolle melk")
	p.Set("image_url", srv.URL+"/541.png")
	if err := a.Save(p); err != nil {
		t.Fatal(err)
	}
	if err := products.KeepPhoto(a, p); err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(p.GetString("thumb"), ".png") {
		t.Fatalf("thumb %q", p.GetString("thumb"))
	}
	if err := products.KeepPhoto(a, p); err != nil || downloads != 1 {
		t.Errorf("kept photo downloaded again: %d, %v", downloads, err)
	}

	p2 := core.NewRecord(col)
	p2.Set("barcode", "5410000000002")
	p2.Set("name", "Afwasmiddel")
	p2.Set("image_url", srv.URL+"/text")
	if err := a.Save(p2); err != nil {
		t.Fatal(err)
	}
	if err := products.KeepPhoto(a, p2); !errors.Is(err, products.ErrNoPhoto) {
		t.Errorf("text as a photo: %v", err)
	}
	p2.Set("image_url", "")
	if err := products.KeepPhoto(a, p2); !errors.Is(err, products.ErrNoPhoto) {
		t.Errorf("no photo: %v", err)
	}
}

func TestKeepLargePhoto(t *testing.T) {
	a := newApp(t)
	var asked []string
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asked = append(asked, r.URL.Path)
		w.Write(png)
	}))
	defer srv.Close()
	products.SetPhotoClient(srv.Client())

	col, _ := a.FindCollectionByNameOrId("products")
	p := core.NewRecord(col)
	p.Set("barcode", "5410000000001")
	p.Set("name", "Halfvolle melk")
	p.Set("image_url", srv.URL+"/541/front_nl.4.200.jpg")
	if err := a.Save(p); err != nil {
		t.Fatal(err)
	}
	if err := products.KeepLargePhoto(a, p); err != nil {
		t.Fatal(err)
	}
	if err := products.KeepLargePhoto(a, p); err != nil {
		t.Fatal(err)
	}
	if len(asked) != 1 || asked[0] != "/541/front_nl.4.400.jpg" {
		t.Errorf("downloaded %v, want the 400 pixel photo once", asked)
	}
	if p.GetString("photo") == "" || p.GetString("thumb") != "" {
		t.Errorf("photo %q, thumb %q", p.GetString("photo"), p.GetString("thumb"))
	}
}
