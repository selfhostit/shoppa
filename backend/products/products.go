// Package products keeps the products collection: the Open Food Facts
// products sold in Belgium, the Netherlands and Greece, from the monthly
// export, and any other product looked up by its barcode. It finds them by
// name in Dutch, French, English or Greek, also through the English category
// words Open Food Facts tags every product with, so "milk" finds "Boni
// halfvolle melk". And it keeps each product's photo at home.
package products

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"github.com/pocketbase/pocketbase/tools/security"
	"github.com/pocketbase/pocketbase/tools/types"
	"github.com/selfhostit/food"
)

// Countries are the ones whose products are kept from the export.
var Countries = []string{"en:belgium", "en:netherlands", "en:greece"}

// UserAgent says who is asking, as Open Food Facts asks apps to; the app sets
// its version in it.
var UserAgent = "Shoppa/dev (self-hosted shopping list)"

// Where products come from; tests point these elsewhere.
var (
	ExportURL = food.OFFExportURL
	Sites     = []Site{
		{"off", food.OpenFoodFacts},
		{"opf", food.OpenProductsFacts},
		{"obf", food.OpenBeautyFacts},
	}
)

// Site is a database a barcode is looked up in.
type Site struct {
	Source string
	URL    string
}

// ErrUnknown means no site knows the barcode.
var ErrUnknown = errors.New("unknown product")

// ImportName is the export's row in product_imports.
const ImportName = "off"

func client() food.Client { return food.Client{UserAgent: UserAgent} }

// Name is a product's name in the language (en, nl or fr), or its main name.
func Name(p *core.Record, lang string) string {
	if lang == "nl" || lang == "fr" {
		if n := p.GetString("name_" + lang); n != "" {
			return n
		}
	}
	return p.GetString("name")
}

// categoryWords turns Open Food Facts' English category tags into words to
// search: "en:dairies,en:semi-skimmed-milks" becomes "dairies semi skimmed
// milks". Tags in other languages are left out.
func categoryWords(tags string) string {
	var words []string
	for _, tag := range strings.Split(tags, ",") {
		if rest, ok := strings.CutPrefix(strings.TrimSpace(tag), "en:"); ok {
			words = append(words, strings.ReplaceAll(rest, "-", " "))
		}
	}
	return strings.Join(words, " ")
}

func searchText(name, nameNL, nameFR, aliases, brand, categories string) string {
	var parts []string
	for _, s := range []string{name, nameNL, nameFR, aliases, brand, categoryWords(categories)} {
		if n := food.Normalize(s); n != "" {
			parts = append(parts, n)
		}
	}
	return truncate(strings.Join(parts, " | "), 3000)
}

// columns are what an import sets; a product keeps its id, created date and
// photo.
var columns = []string{"barcode", "name", "name_nl", "name_fr", "brand", "quantity", "categories",
	"image_url", "search", "source"}

func params(f food.Food, source string) dbx.Params {
	p := dbx.Params{
		"barcode":    f.Barcode,
		"name":       truncate(strings.TrimSpace(f.Name), 200),
		"name_nl":    truncate(strings.TrimSpace(f.NameNL), 200),
		"name_fr":    truncate(strings.TrimSpace(f.NameFR), 200),
		"brand":      truncate(strings.TrimSpace(f.Brand), 200),
		"quantity":   truncate(strings.TrimSpace(f.Quantity), 50),
		"categories": truncate(f.Categories, 2000),
		"image_url":  truncate(f.ImageURL, 500),
		"source":     source,
	}
	p["search"] = searchText(f.Name, f.NameNL, f.NameFR, f.Aliases, f.Brand, f.Categories)
	return p
}

// Store adds the products, or updates the ones already kept, by barcode.
func Store(app core.App, source string, products []food.Food) error {
	set := make([]string, len(columns))
	names := []string{"[[id]]", "[[created]]", "[[updated]]"}
	values := []string{"{:id}", "{:created}", "{:updated}"}
	for i, c := range columns {
		set[i] = fmt.Sprintf("[[%s]] = excluded.[[%s]]", c, c)
		names = append(names, "[["+c+"]]")
		values = append(values, "{:"+c+"}")
	}
	// raw SQL skips PocketBase's autodate fields, so the dates are set here
	sql := fmt.Sprintf("INSERT INTO {{products}} (%s) VALUES (%s) ON CONFLICT ([[barcode]]) DO UPDATE SET %s, [[updated]] = excluded.[[updated]]",
		strings.Join(names, ", "), strings.Join(values, ", "), strings.Join(set, ", "))
	return app.RunInTransaction(func(tx core.App) error {
		now := types.NowDateTime().String()
		for _, f := range products {
			p := params(f, source)
			if p["name"] == "" || !food.BarcodePattern.MatchString(f.Barcode) {
				continue
			}
			p["id"] = security.RandomStringWithAlphabet(15, "abcdefghijklmnopqrstuvwxyz0123456789")
			p["created"], p["updated"] = now, now
			if _, err := tx.DB().NewQuery(sql).Bind(p).Execute(); err != nil {
				return fmt.Errorf("product %s: %w", f.Barcode, err)
			}
		}
		return nil
	})
}

const batchSize = 500

var importing sync.Mutex

// ErrBusy means the export is already being imported.
var ErrBusy = errors.New("the products are already being imported")

// Import downloads the Open Food Facts export (about a gigabyte, read as it
// streams in) and keeps the products sold in the Countries. Its progress and
// result are in product_imports.
func Import(app core.App) error {
	if !importing.TryLock() {
		return ErrBusy
	}
	defer importing.Unlock()

	status, err := importStatus(app)
	if err != nil {
		return err
	}
	status.Set("running", true)
	status.Set("error", "")
	if err := app.Save(status); err != nil {
		return err
	}

	err = func() error {
		body, err := client().Export(context.Background(), ExportURL)
		if err != nil {
			return err
		}
		defer body.Close()
		var batch []food.Food
		err = food.ReadExport(body, Countries, func(f food.Food) error {
			if batch = append(batch, f); len(batch) < batchSize {
				return nil
			}
			err := Store(app, "off", batch)
			batch = batch[:0]
			return err
		})
		if err != nil {
			return err
		}
		return Store(app, "off", batch)
	}()

	status.Set("running", false)
	if err != nil {
		status.Set("error", truncate(err.Error(), 500))
	} else {
		n, _ := app.CountRecords("products", dbx.HashExp{"source": "off"})
		status.Set("products", n)
		status.Set("version", time.Now().Format("2006-01-02"))
		status.Set("imported", types.NowDateTime())
	}
	if saveErr := app.Save(status); saveErr != nil && err == nil {
		err = saveErr
	}
	return err
}

func importStatus(app core.App) (*core.Record, error) {
	if r, err := app.FindFirstRecordByData("product_imports", "name", ImportName); err == nil {
		return r, nil
	}
	col, err := app.FindCollectionByNameOrId("product_imports")
	if err != nil {
		return nil, err
	}
	r := core.NewRecord(col)
	r.Set("name", ImportName)
	return r, nil
}

// Lookup finds a product by barcode: one already kept, or else one looked up
// in Open Food Facts, Open Products Facts and Open Beauty Facts, and kept.
func Lookup(app core.App, code string) (*core.Record, error) {
	if !food.BarcodePattern.MatchString(code) {
		return nil, ErrUnknown
	}
	codes := []any{code}
	if len(code) == 12 {
		// a UPC-A code is the EAN-13 code with a leading zero
		codes = append(codes, "0"+code)
	} else if len(code) == 13 && code[0] == '0' {
		codes = append(codes, code[1:])
	}
	var found []*core.Record
	if err := app.RecordQuery("products").AndWhere(dbx.In("barcode", codes...)).Limit(1).All(&found); err != nil {
		return nil, err
	}
	if len(found) > 0 {
		return found[0], nil
	}

	var lastErr error = ErrUnknown
	for _, site := range Sites {
		f, err := client().Product(context.Background(), site.URL, code)
		if errors.Is(err, food.ErrUnknownProduct) {
			continue
		}
		if err != nil {
			lastErr = fmt.Errorf("%s: %w", site.Source, err)
			continue
		}
		f.Barcode = code
		if err := Store(app, site.Source, []food.Food{f}); err != nil {
			return nil, err
		}
		return app.FindFirstRecordByData("products", "barcode", code)
	}
	return nil, lastErr
}

var photoClient = &http.Client{Timeout: 15 * time.Second}

// maxPhoto is the largest photo kept; the small ones are a few tens of kB, the
// large ones about a hundred.
const maxPhoto = 1 << 20

// ErrNoPhoto means the product has no photo, or it can't be downloaded.
var ErrNoPhoto = errors.New("no photo")

// KeepPhoto downloads the product's small photo into its thumb field, once.
func KeepPhoto(app core.App, p *core.Record) error {
	return keep(app, p, "thumb", p.GetString("image_url"))
}

// smallSize is how Open Food Facts names its small photos: front_nl.4.200.jpg
// is 200 pixels high, and front_nl.4.400.jpg the same photo at 400.
var smallSize = regexp.MustCompile(`\.200\.(jpg|png|webp)$`)

// KeepLargePhoto downloads the product's photo at 400 pixels into its photo
// field, once, for seeing it better. A product whose photo comes in one size
// only keeps that one again.
func KeepLargePhoto(app core.App, p *core.Record) error {
	return keep(app, p, "photo", smallSize.ReplaceAllString(p.GetString("image_url"), ".400.$1"))
}

func keep(app core.App, p *core.Record, field, url string) error {
	if p.GetString(field) != "" {
		return nil
	}
	if !strings.HasPrefix(url, "https://") {
		return ErrNoPhoto
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", UserAgent)
	res, err := photoClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: status %d", ErrNoPhoto, res.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(res.Body, maxPhoto+1))
	if err != nil {
		return err
	}
	ext := map[string]string{"image/jpeg": ".jpg", "image/png": ".png", "image/webp": ".webp"}[http.DetectContentType(data)]
	if len(data) > maxPhoto || ext == "" {
		return ErrNoPhoto
	}
	file, err := filesystem.NewFileFromBytes(data, p.GetString("barcode")+ext)
	if err != nil {
		return err
	}
	p.Set(field, file)
	return app.Save(p)
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}
