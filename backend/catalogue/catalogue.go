// Package catalogue holds the generic items Shoppa ships with ("melk",
// "bananen", "wc-papier", "sofa", "tent"), each with a Dutch, French, English
// and Greek name, a kind of list, a section and an icon, and keeps the
// generics collection in step with it.
package catalogue

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/pocketbase/pocketbase/core"
	"github.com/selfhostit/food"
)

//go:embed generics.json
var data []byte

// The kinds of list. Each has its own sections and items.
const (
	Groceries = "groceries"
	Home      = "home"
	Camping   = "camping"
	Trip      = "trip"
)

// Kinds is every kind of list, groceries first.
var Kinds = []string{Groceries, Home, Camping, Trip}

// Section is a part of the store, or of a list of another kind; the list is
// grouped by them, in this order. Other has no kind: every list has it.
type Section struct {
	Key  string `json:"key"`
	Kind string `json:"kind,omitempty"`
	// a line icon's name (web/src/lib/icons.ts)
	Icon  string `json:"icon"`
	Emoji string `json:"emoji,omitempty"`
	EN    string `json:"en"`
	NL    string `json:"nl"`
	FR    string `json:"fr"`
	EL    string `json:"el"`
}

// Item is one generic item.
type Item struct {
	Slug string `json:"slug"`
	// the kind of list it is for; groceries when empty
	Kind    string `json:"kind,omitempty"`
	Section string `json:"section"`
	Icon    string `json:"icon"`
	Emoji   string `json:"emoji,omitempty"`
	EN      string `json:"en"`
	NL      string `json:"nl"`
	FR      string `json:"fr"`
	EL      string `json:"el"`
	// other names people use, separated by semicolons
	Aliases string `json:"aliases,omitempty"`
	// Open Food Facts category tags, comma-separated, for suggesting products
	OFF string `json:"off,omitempty"`
}

// Catalogue is the bundled sections and items.
type Catalogue struct {
	Sections []Section `json:"sections"`
	Items    []Item    `json:"items"`
}

// Load reads the bundled catalogue.
func Load() (Catalogue, error) {
	var c Catalogue
	if err := json.Unmarshal(data, &c); err != nil {
		return c, fmt.Errorf("reading the catalogue: %w", err)
	}
	for i := range c.Items {
		if c.Items[i].Kind == "" {
			c.Items[i].Kind = Groceries
		}
	}
	return c, nil
}

// ValidKind tells whether kind is one of the kinds of list.
func ValidKind(kind string) bool {
	return slices.Contains(Kinds, kind)
}

// SectionFits tells whether a section is one a list of the kind has.
func (c Catalogue) SectionFits(key, kind string) bool {
	for _, s := range c.Sections {
		if s.Key == key {
			return s.Kind == "" || s.Kind == kind
		}
	}
	return false
}

// Other is the section for things that aren't in any other.
const Other = "other"

// Names joins every name of a generic, normalized, between bars, so a search
// for a whole name is a LIKE on "|melk|".
func Names(names ...string) string {
	var parts []string
	seen := map[string]bool{}
	for _, n := range names {
		for _, part := range strings.Split(n, ";") {
			if s := food.Normalize(part); s != "" && !seen[s] {
				seen[s] = true
				parts = append(parts, s)
			}
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return "|" + strings.Join(parts, "|") + "|"
}

// SetNames fills a generic record's names field from its names and aliases.
func SetNames(r *core.Record) {
	r.Set("names", Names(r.GetString("name_en"), r.GetString("name_nl"), r.GetString("name_fr"),
		r.GetString("name_el"), r.GetString("aliases")))
}

// Sync adds the bundled items to the generics collection and updates the ones
// it has, matched by slug. Generics people added themselves are left alone.
func Sync(app core.App) error {
	c, err := Load()
	if err != nil {
		return err
	}
	col, err := app.FindCollectionByNameOrId("generics")
	if err != nil {
		return err
	}
	existing, err := app.FindAllRecords(col)
	if err != nil {
		return err
	}
	bySlug := make(map[string]*core.Record, len(existing))
	for _, r := range existing {
		bySlug[r.GetString("slug")] = r
	}
	return app.RunInTransaction(func(tx core.App) error {
		for _, it := range c.Items {
			r := bySlug[it.Slug]
			if r == nil {
				r = core.NewRecord(col)
				r.Set("slug", it.Slug)
			} else if r.GetBool("custom") {
				continue
			}
			before := r.PublicExport()
			r.Set("kind", it.Kind)
			r.Set("section", it.Section)
			r.Set("icon", it.Icon)
			r.Set("emoji", it.Emoji)
			r.Set("name_en", it.EN)
			r.Set("name_nl", it.NL)
			r.Set("name_fr", it.FR)
			r.Set("name_el", it.EL)
			r.Set("aliases", it.Aliases)
			r.Set("off_categories", it.OFF)
			SetNames(r)
			if !r.IsNew() && same(before, r.PublicExport()) {
				continue
			}
			if err := tx.Save(r); err != nil {
				return fmt.Errorf("generic %s: %w", it.Slug, err)
			}
		}
		return nil
	})
}

func same(a, b map[string]any) bool {
	for k, v := range b {
		if fmt.Sprint(a[k]) != fmt.Sprint(v) {
			return false
		}
	}
	return true
}
