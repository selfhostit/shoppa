package catalogue

import (
	"strings"
	"testing"
)

func TestCatalogue(t *testing.T) {
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	sections := map[string]bool{}
	perKind := map[string]int{}
	for _, s := range c.Sections {
		if sections[s.Key] || s.EN == "" || s.NL == "" || s.FR == "" || s.EL == "" || s.Icon == "" ||
			(s.Kind == "") != (s.Key == Other) || (s.Kind != "" && !ValidKind(s.Kind)) {
			t.Errorf("section %+v", s)
		}
		sections[s.Key] = true
		perKind[s.Kind]++
	}
	if !sections[Other] || c.Sections[len(c.Sections)-1].Key != Other {
		t.Error("other isn't the last section")
	}
	for _, kind := range Kinds {
		if perKind[kind] < 5 {
			t.Errorf("%d sections for %s", perKind[kind], kind)
		}
	}
	if len(c.Items) < 200 {
		t.Errorf("only %d items", len(c.Items))
	}
	slugs := map[string]bool{}
	for _, it := range c.Items {
		if slugs[it.Slug] || !c.SectionFits(it.Section, it.Kind) || it.Section == Other || it.EN == "" || it.NL == "" || it.FR == "" ||
			it.EL == "" || it.Icon == "" ||
			strings.HasPrefix(it.Slug, "custom-") {
			t.Errorf("item %+v", it)
		}
		slugs[it.Slug] = true
	}
}

func TestNames(t *testing.T) {
	if got := Names("Milk", "Melk", "Lait", "halfvolle melk; Lait demi-écrémé"); got != "|milk|melk|lait|halfvolle melk|lait demi ecreme|" {
		t.Errorf("names %q", got)
	}
	if Names("", " ") != "" {
		t.Error("no names")
	}
}
