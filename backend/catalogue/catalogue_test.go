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
	for _, s := range c.Sections {
		if sections[s.Key] || s.EN == "" || s.NL == "" || s.FR == "" || s.EL == "" || s.Emoji == "" {
			t.Errorf("section %+v", s)
		}
		sections[s.Key] = true
	}
	if !sections[Other] {
		t.Error("no other section")
	}
	if len(c.Items) < 200 {
		t.Errorf("only %d items", len(c.Items))
	}
	slugs := map[string]bool{}
	for _, it := range c.Items {
		if slugs[it.Slug] || !sections[it.Section] || it.EN == "" || it.NL == "" || it.FR == "" || it.EL == "" || it.Emoji == "" ||
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
