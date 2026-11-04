package products

import (
	"slices"
	"sort"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/selfhostit/food"
)

// how many matches are ranked; the rest are left out
const candidates = 300

// a search: the words that were typed, or another way to say them
type phrasing struct {
	words []string
	// typed, rather than a translation from the catalogue
	typed bool
}

// Search finds products for what was typed, best first. A product matches
// when its names, brand or English category words hold every word, in any
// order and without minding accents. A word that names a catalogue item
// searches its names in the other languages too, and its Open Food Facts
// categories: "cheese" finds "Jonge Gouda kaas", "lait" finds "Halfvolle melk".
func Search(app core.App, q, lang string, limit int) ([]*core.Record, error) {
	words := food.Words(q)
	if len(words) == 0 {
		return []*core.Record{}, nil
	}
	phrasings := []phrasing{{words, true}}
	if stems := food.Stemmed(words); stems != nil {
		phrasings = append(phrasings, phrasing{stems, true})
	}
	var tags []string
	for _, g := range genericsNamed(app, words) {
		for _, n := range []string{g.GetString("name_en"), g.GetString("name_nl"), g.GetString("name_fr")} {
			if w := food.Words(n); len(w) > 0 && !slices.ContainsFunc(phrasings, func(p phrasing) bool { return slices.Equal(p.words, w) }) {
				phrasings = append(phrasings, phrasing{w, false})
			}
		}
		for _, t := range strings.Split(g.GetString("off_categories"), ",") {
			if t = strings.TrimSpace(t); t != "" && !slices.Contains(tags, t) {
				tags = append(tags, t)
			}
		}
	}

	var either []dbx.Expression
	for _, p := range phrasings {
		var all []dbx.Expression
		for _, w := range p.words {
			all = append(all, dbx.Like("search", w))
		}
		either = append(either, dbx.And(all...))
	}
	for i, t := range tags {
		key := "tag" + string(rune('a'+i))
		either = append(either, dbx.NewExp("(',' || [[categories]] || ',') LIKE {:"+key+"}", dbx.Params{key: "%," + t + ",%"}))
	}
	var found []*core.Record
	err := app.RecordQuery("products").
		AndWhere(dbx.Or(either...)).
		// products with a photo and short names first, so the best are among
		// the candidates
		OrderBy("(image_url != '') DESC", "length(name)").
		Limit(candidates).
		All(&found)
	if err != nil {
		return nil, err
	}

	scores := make(map[string]float64, len(found))
	for _, r := range found {
		scores[r.Id] = score(r, phrasings, tags, lang)
	}
	sort.SliceStable(found, func(i, j int) bool {
		if scores[found[i].Id] != scores[found[j].Id] {
			return scores[found[i].Id] > scores[found[j].Id]
		}
		return found[i].GetString("name") < found[j].GetString("name")
	})
	if len(found) > limit {
		found = found[:limit]
	}
	return found, nil
}

// genericsNamed finds the catalogue items whose name, in any language, is
// what was typed, or starts with it once three letters are typed.
func genericsNamed(app core.App, words []string) []*core.Record {
	q := strings.Join(words, " ")
	exp := dbx.Like("names", "|"+q+"|")
	if len([]rune(q)) >= 3 {
		exp = dbx.Like("names", "|"+q)
	}
	var found []*core.Record
	if err := app.RecordQuery("generics").AndWhere(exp).OrderBy("length(name_en)").Limit(3).All(&found); err != nil {
		return nil
	}
	return found
}

// score rates how well a product matches: its names first, a brand adds to
// them, then its categories; what was typed counts more than a translation of
// it, and a name in the person's language more than another.
func score(r *core.Record, phrasings []phrasing, tags []string, lang string) float64 {
	best := 0.0
	for _, p := range phrasings {
		weight := 1.0
		if !p.typed {
			weight = 0.8
		}
		for _, field := range []string{"name", "name_nl", "name_fr"} {
			s := food.FieldScore(food.Normalize(r.GetString(field)), p.words)
			if field == "name_"+lang {
				s *= 1.1
			}
			best = max(best, s*weight)
		}
		brand := food.FieldScore(food.Normalize(r.GetString("brand")), p.words)
		cats := food.FieldScore(food.Normalize(categoryWords(r.GetString("categories"))), p.words)
		best = max(best, (0.5*brand+0.6*cats)*weight)
	}
	total := best
	cats := "," + r.GetString("categories") + ","
	for _, t := range tags {
		if strings.Contains(cats, ","+t+",") {
			total += 3
			break
		}
	}
	if r.GetString("image_url") != "" {
		total += 0.5
	}
	// "Melk" before "Melk met een vleugje vanille en kaneel"
	return total - 0.1*float64(len(strings.Fields(r.GetString("name"))))
}
