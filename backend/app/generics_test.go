package app_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

// withLang sets a person's language.
func withLang(t *testing.T, a core.App, user *core.Record, lang string) {
	user.Set("lang", lang)
	if err := a.Save(user); err != nil {
		t.Fatal(err)
	}
}

func TestTheCatalogueIsLoaded(t *testing.T) {
	a := newApp(t)
	defer a.Cleanup()
	_, token := addUser(t, a, "ada@example.com", "owner")
	got := send(t, a, http.StatusOK, http.MethodGet, "/api/collections/generics/records?filter=slug='milk'", token, "")
	items, _ := got["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("milk %v", got)
	}
	milk := items[0].(map[string]any)
	if milk["name_nl"] != "Melk" || milk["name_fr"] != "Lait" || milk["section"] != "dairy" || milk["custom"] != false {
		t.Errorf("milk %v", milk)
	}
	if n, _ := a.CountRecords("generics"); n < 200 {
		t.Errorf("%d generics", n)
	}

	s := tests.ApiScenario{
		Method:                http.MethodGet,
		URL:                   "/api/app/sections",
		ExpectedStatus:        http.StatusOK,
		ExpectedContent:       []string{`"key":"produce"`, `"nl":"Groenten & fruit"`},
		TestAppFactory:        func(testing.TB) *tests.TestApp { return a },
		DisableTestAppCleanup: true,
	}
	s.Test(t)
}

func TestTypedNamesBecomeGenerics(t *testing.T) {
	a := newApp(t)
	defer a.Cleanup()
	ada, adaToken := addUser(t, a, "ada@example.com", "owner")
	bob, bobToken := addUser(t, a, "bob@example.com", "member")
	withLang(t, a, ada, "en")
	withLang(t, a, bob, "fr")
	list := newList(t, a, "Weekend")
	add := func(want int, token, body string) map[string]any {
		t.Helper()
		return send(t, a, want, http.MethodPost, "/api/app/items", token, body)
	}

	// "melk" is the generic milk, saved in the adder's language
	milk := add(http.StatusCreated, adaToken, `{"list":"`+list+`","name":"melk"}`)
	if milk["name"] != "Milk" || milk["generic"] == "" {
		t.Fatalf("milk %v", milk)
	}
	// "Lait" is the same thing
	lait := add(http.StatusOK, bobToken, `{"list":"`+list+`","name":"Lait","quantity":"2×"}`)
	if lait["id"] != milk["id"] || lait["quantity"] != "2×" {
		t.Fatalf("lait %v", lait)
	}
	// and so is the generic itself
	again := add(http.StatusOK, bobToken, `{"list":"`+list+`","generic":"`+milk["generic"].(string)+`"}`)
	if again["id"] != milk["id"] {
		t.Fatalf("again %v", again)
	}

	// a name in two languages is the one in your own: raisins are grapes in French
	grapes := add(http.StatusCreated, bobToken, `{"list":"`+list+`","name":"raisins"}`)
	raisins := add(http.StatusCreated, adaToken, `{"list":"`+list+`","name":"raisins"}`)
	if grapes["name"] != "Raisins" || raisins["name"] != "Raisins" || grapes["generic"] == raisins["generic"] {
		t.Errorf("grapes %v, raisins %v", grapes, raisins)
	}

	// something not in the catalogue stays what was typed
	if free := add(http.StatusCreated, adaToken, `{"list":"`+list+`","name":"Kattengras"}`); free["generic"] != "" || free["name"] != "Kattengras" {
		t.Errorf("free %v", free)
	}
	add(http.StatusNotFound, adaToken, `{"list":"`+list+`","generic":"nosuchgeneric12"}`)

	// buying it keeps the generic with the purchase
	url := "/api/collections/items/records/" + milk["id"].(string)
	send(t, a, http.StatusOK, http.MethodPatch, url, adaToken, `{"checked":true}`)
	if p, err := a.FindFirstRecordByData("purchases", "item", milk["id"]); err != nil || p.GetString("generic") != milk["generic"] {
		t.Errorf("purchase %v, %v", p, err)
	}

	// renaming it to something else, without the generic, makes it free text
	renamed := send(t, a, http.StatusOK, http.MethodPatch, url, adaToken, `{"name":"Havermelk van de boer","generic":""}`)
	if renamed["generic"] != "" {
		t.Errorf("renamed %v", renamed)
	}
	if again := add(http.StatusCreated, adaToken, `{"list":"`+list+`","name":"milk"}`); again["id"] == milk["id"] {
		t.Errorf("milk came back as the renamed item")
	}
}

func TestPeopleAddTheirOwnGenerics(t *testing.T) {
	a := newApp(t)
	defer a.Cleanup()
	_, token := addUser(t, a, "ada@example.com", "owner")
	list := newList(t, a, "Weekend")

	own := send(t, a, http.StatusOK, http.MethodPost, "/api/collections/generics/records", token,
		`{"name_nl":"Kattengras","emoji":"🌱","section":"nowhere","custom":false,"slug":"milk"}`)
	slug, _ := own["slug"].(string)
	if own["custom"] != true || !strings.HasPrefix(slug, "custom-kattengras-") || own["section"] != "other" ||
		own["name_en"] != "Kattengras" || own["names"] != "|kattengras|" {
		t.Fatalf("own %v", own)
	}
	if item := send(t, a, http.StatusCreated, http.MethodPost, "/api/app/items", token,
		`{"list":"`+list+`","name":"kattengras"}`); item["generic"] != own["id"] {
		t.Errorf("item %v", item)
	}

	// your own can change; the bundled ones can't
	changed := send(t, a, http.StatusOK, http.MethodPatch, "/api/collections/generics/records/"+own["id"].(string), token,
		`{"section":"pets","slug":"other"}`)
	if changed["section"] != "pets" || changed["slug"] != slug {
		t.Errorf("changed %v", changed)
	}
	milk, err := a.FindFirstRecordByData("generics", "slug", "milk")
	if err != nil {
		t.Fatal(err)
	}
	send(t, a, http.StatusNotFound, http.MethodPatch, "/api/collections/generics/records/"+milk.Id, token, `{"emoji":"🐄"}`)
	send(t, a, http.StatusNotFound, http.MethodDelete, "/api/collections/generics/records/"+milk.Id, token, "")
}
