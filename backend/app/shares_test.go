package app_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/tests"
)

func TestSharingAListWithAGuest(t *testing.T) {
	a := newApp(t)
	defer a.Cleanup()
	ada, token := addUser(t, a, "ada@example.com", "owner")
	list := newList(t, a, "Groceries")
	other := newList(t, a, "Camping")
	melk := send(t, a, http.StatusCreated, http.MethodPost, "/api/app/items", token, `{"list":"`+list+`","name":"Melk","quantity":"2×"}`)
	tent := send(t, a, http.StatusCreated, http.MethodPost, "/api/app/items", token, `{"list":"`+other+`","name":"Tent"}`)

	view := send(t, a, http.StatusOK, http.MethodPost, "/api/collections/shares/records", token,
		`{"list":"`+list+`","token":"mine","by":""}`)
	if tok := view["token"].(string); len(tok) != 32 || view["by"] != ada.Id {
		t.Fatalf("share %v", view)
	}
	guest := func(want int, method, url, body string) map[string]any {
		t.Helper()
		return send(t, a, want, method, url, "", body)
	}
	viewURL := "/api/app/shared/" + view["token"].(string)

	// the guest sees the list, in their language, and nothing about people
	got := guest(http.StatusOK, http.MethodGet, viewURL+"?lang=fr", "")
	items := got["items"].([]any)
	if got["name"] != "Groceries" || got["can_tick"] != false || len(items) != 1 {
		t.Fatalf("shared %v", got)
	}
	if it := items[0].(map[string]any); it["name"] != "Lait" || it["quantity"] != "2×" || it["added_by"] != nil {
		t.Errorf("item %v", it)
	}
	// a view-only link doesn't tick
	guest(http.StatusForbidden, http.MethodPost, viewURL+"/items/"+melk["id"].(string), `{"checked":true}`)

	tick := send(t, a, http.StatusOK, http.MethodPost, "/api/collections/shares/records", token,
		`{"list":"`+list+`","can_tick":true}`)
	tickURL := "/api/app/shared/" + tick["token"].(string)
	noContent(t, a, http.MethodPost, tickURL+"/items/"+melk["id"].(string), `{"checked":true}`)
	if r, _ := a.FindRecordById("items", melk["id"].(string)); !r.GetBool("checked") {
		t.Error("the guest's tick didn't stick")
	}
	if n, _ := a.CountRecords("purchases"); n != 1 {
		t.Errorf("%d purchases", n)
	}
	// only on the shared list, with a real token
	guest(http.StatusNotFound, http.MethodPost, tickURL+"/items/"+tent["id"].(string), `{"checked":true}`)
	guest(http.StatusNotFound, http.MethodGet, "/api/app/shared/nope", "")

	// deleting the share stops the link
	r, _ := a.FindRecordById("shares", view["id"].(string))
	if err := a.Delete(r); err != nil {
		t.Fatal(err)
	}
	guest(http.StatusNotFound, http.MethodGet, viewURL, "")
}

func noContent(t *testing.T, a *tests.TestApp, method, url, body string) {
	t.Helper()
	(&tests.ApiScenario{
		Method: method, URL: url, Body: strings.NewReader(body),
		ExpectedStatus:        http.StatusNoContent,
		TestAppFactory:        func(testing.TB) *tests.TestApp { return a },
		DisableTestAppCleanup: true,
	}).Test(t)
}
