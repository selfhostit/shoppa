package app

import (
	"net/http"
	"strings"
	"sync"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/security"

	"github.com/selfhostit/food"
	"github.com/selfhostit/shoppa/catalogue"
)

func registerGenerics(app core.App) {
	// the bundled catalogue is brought up to date each time the server starts
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		if err := catalogue.Sync(se.App); err != nil {
			se.App.Logger().Error("updating the catalogue", "error", err)
		}
		return se.Next()
	})

	// a generic someone adds is theirs to change: custom, with its own slug,
	// a section, and its one name in every language it wasn't given in
	app.OnRecordCreateRequest("generics").BindFunc(func(e *core.RecordRequestEvent) error {
		r := e.Record
		r.Set("custom", true)
		name := firstOf(r.GetString("name_en"), r.GetString("name_nl"), r.GetString("name_fr"), r.GetString("name_el"))
		// a Greek name has no letters a slug can hold: "custom-x7k2"
		slug := "custom-"
		if s := slugOf(name); s != "" {
			slug += s + "-"
		}
		r.Set("slug", slug+security.RandomStringWithAlphabet(4, "abcdefghijklmnopqrstuvwxyz0123456789"))
		for _, f := range []string{"name_en", "name_nl", "name_fr", "name_el"} {
			if strings.TrimSpace(r.GetString(f)) == "" {
				r.Set(f, name)
			}
		}
		return e.Next()
	})
	app.OnRecordUpdateRequest("generics").BindFunc(func(e *core.RecordRequestEvent) error {
		e.Record.Set("custom", true)
		e.Record.Set("slug", e.Record.Original().GetString("slug"))
		return e.Next()
	})
	setNames := func(e *core.RecordEvent) error {
		if !catalogue.ValidKind(e.Record.GetString("kind")) {
			e.Record.Set("kind", catalogue.Groceries)
		}
		if !validSection(e.Record.GetString("section"), e.Record.GetString("kind")) {
			e.Record.Set("section", catalogue.Other)
		}
		catalogue.SetNames(e.Record)
		return e.Next()
	}
	app.OnRecordCreate("generics").BindFunc(setNames)
	app.OnRecordUpdate("generics").BindFunc(setNames)
}

func registerGenericRoutes(se *core.ServeEvent) {
	se.Router.GET("/api/app/sections", sections)
}

// sections tells the web app the sections of every kind of list, in the
// order the list shows them.
func sections(e *core.RequestEvent) error {
	c, err := catalogue.Load()
	if err != nil {
		return err
	}
	return e.JSON(http.StatusOK, c.Sections)
}

// validSection tells whether a section is one of a kind of list.
func validSection(key, kind string) bool {
	c, err := bundled()
	return err == nil && c.SectionFits(key, kind)
}

// bundled is the bundled catalogue, read once: every generic saved checks
// its section against it.
var bundled = sync.OnceValues(catalogue.Load)

// listKind is the kind of a list: groceries, home, camping or trip.
func listKind(app core.App, id string) string {
	if l, err := app.FindRecordById("lists", id); err == nil && catalogue.ValidKind(l.GetString("kind")) {
		return l.GetString("kind")
	}
	return catalogue.Groceries
}

// matchGeneric finds the generic of a kind of list named exactly what was
// typed (normalized), in any language. When names clash across languages
// ("raisins" is French for grapes), the one in the person's language wins.
func matchGeneric(app core.App, key, lang, kind string) *core.Record {
	if key == "" {
		return nil
	}
	var found []*core.Record
	err := app.RecordQuery("generics").
		AndWhere(dbx.HashExp{"kind": kind}).
		AndWhere(dbx.Like("names", "|"+key+"|").Match(true, true)).
		OrderBy("custom DESC", "created").
		Limit(10).
		All(&found)
	if err != nil || len(found) == 0 {
		return nil
	}
	for _, g := range found {
		if food.Normalize(genericName(g, lang)) == key {
			return g
		}
	}
	return found[0]
}

// genericName is a generic's name in the language (en, nl, fr or el).
func genericName(g *core.Record, lang string) string {
	if lang != "nl" && lang != "fr" && lang != "el" {
		lang = "en"
	}
	return firstOf(g.GetString("name_"+lang), g.GetString("name_en"), g.GetString("name_nl"),
		g.GetString("name_fr"), g.GetString("name_el"))
}

// personLang is the language a person sees item names in; Dutch until they
// pick one.
func personLang(auth *core.Record) string {
	if auth != nil && auth.Collection().Name == "users" {
		if lang := auth.GetString("lang"); lang != "" {
			return lang
		}
	}
	return "nl"
}

// slugOf is a name's letters a to z and digits, with dashes between words.
func slugOf(s string) string {
	words := strings.FieldsFunc(food.Normalize(s), func(r rune) bool {
		return (r < 'a' || r > 'z') && (r < '0' || r > '9')
	})
	return strings.Join(words, "-")
}

func firstOf(values ...string) string {
	for _, v := range values {
		if v = strings.TrimSpace(v); v != "" {
			return v
		}
	}
	return ""
}
