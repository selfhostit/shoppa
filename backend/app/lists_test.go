package app_test

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

// newList saves a list and returns its id.
func newList(t testing.TB, a core.App, name string) string {
	lists, err := a.FindCollectionByNameOrId("lists")
	if err != nil {
		t.Fatal(err)
	}
	l := core.NewRecord(lists)
	l.Set("name", name)
	if err := a.Save(l); err != nil {
		t.Fatal(err)
	}
	return l.Id
}

// send makes a JSON request as a person, checks its status and returns the
// body.
func send(t *testing.T, a *tests.TestApp, want int, method, url, token, body string) map[string]any {
	t.Helper()
	var got map[string]any
	s := tests.ApiScenario{
		Method:          method,
		URL:             url,
		Body:            strings.NewReader(body),
		Headers:         map[string]string{"Authorization": token},
		ExpectedStatus:  want,
		ExpectedContent: []string{"{"},
		TestAppFactory:  func(testing.TB) *tests.TestApp { return a },
		AfterTestFunc: func(t testing.TB, _ *tests.TestApp, res *http.Response) {
			data, _ := io.ReadAll(res.Body)
			_ = json.Unmarshal(data, &got)
		},
		DisableTestAppCleanup: true,
	}
	s.Test(t)
	return got
}

func TestTheFirstPersonGetsAList(t *testing.T) {
	a := newApp(t)
	defer a.Cleanup()
	addUser(t, a, "ada@example.com", "owner")
	addUser(t, a, "bob@example.com", "member")
	lists, err := a.FindAllRecords("lists")
	if err != nil {
		t.Fatal(err)
	}
	if len(lists) != 1 || lists[0].GetString("name") != "Groceries" {
		t.Fatalf("lists %v", lists)
	}
}

func TestAddingAnItemTwiceKeepsOne(t *testing.T) {
	a := newApp(t)
	defer a.Cleanup()
	ada, adaToken := addUser(t, a, "ada@example.com", "owner")
	bob, bobToken := addUser(t, a, "bob@example.com", "member")
	list := newList(t, a, "Weekend")
	add := func(want int, token, body string) map[string]any {
		t.Helper()
		return send(t, a, want, http.MethodPost, "/api/app/items", token, body)
	}

	melk := add(http.StatusCreated, adaToken, `{"list":"`+list+`","name":"  Verse   boerenmelk ","quantity":"2×"}`)
	if melk["name"] != "Verse boerenmelk" || melk["quantity"] != "2×" || melk["added_by"] != ada.Id || melk["checked"] != false {
		t.Fatalf("melk %v", melk)
	}
	if _, sent := melk["key"]; sent {
		t.Error("the key is sent to phones")
	}

	// the same thing, written differently, is the same item; a note sent along is kept
	again := add(http.StatusOK, bobToken, `{"list":"`+list+`","name":"verse BOERENMÉLK","note":"the big one"}`)
	if again["id"] != melk["id"] || again["note"] != "the big one" || again["quantity"] != "2×" || again["added_by"] != ada.Id {
		t.Fatalf("again %v", again)
	}
	if n, _ := a.CountRecords("items"); n != 1 {
		t.Fatalf("%d items", n)
	}

	// ticked off, it comes back from recently bought, added by whoever brought it back
	item, err := a.FindRecordById("items", melk["id"].(string))
	if err != nil {
		t.Fatal(err)
	}
	item.Set("checked", true)
	if err := a.Save(item); err != nil {
		t.Fatal(err)
	}
	back := add(http.StatusOK, bobToken, `{"list":"`+list+`","name":"Verse boerenmelk"}`)
	if back["id"] != melk["id"] || back["checked"] != false || back["added_by"] != bob.Id || back["note"] != "the big one" {
		t.Fatalf("back %v", back)
	}

	// the same name on another list is another item
	party := add(http.StatusCreated, bobToken, `{"list":"`+newList(t, a, "Party")+`","name":"Verse boerenmelk"}`)
	if party["id"] == melk["id"] {
		t.Fatalf("party %v", party)
	}
}

func TestAddingAnItem(t *testing.T) {
	a := newApp(t)
	defer a.Cleanup()
	_, token := addUser(t, a, "ada@example.com", "owner")
	list := newList(t, a, "Weekend")

	// a phone that made the item offline keeps its id
	made := send(t, a, http.StatusCreated, http.MethodPost, "/api/app/items", token,
		`{"id":"abcdefghij12345","list":"`+list+`","name":"Brood"}`)
	if made["id"] != "abcdefghij12345" {
		t.Fatalf("made %v", made)
	}

	send(t, a, http.StatusBadRequest, http.MethodPost, "/api/app/items", token, `{"list":"`+list+`","name":"  - "}`)
	send(t, a, http.StatusBadRequest, http.MethodPost, "/api/app/items", token, `{"id":"../x","list":"`+list+`","name":"Kaas"}`)
	send(t, a, http.StatusNotFound, http.MethodPost, "/api/app/items", token, `{"list":"nolist","name":"Kaas"}`)
	send(t, a, http.StatusUnauthorized, http.MethodPost, "/api/app/items", "", `{"list":"`+list+`","name":"Kaas"}`)
	// items only come in through the route, which finds the one already there
	send(t, a, http.StatusForbidden, http.MethodPost, "/api/collections/items/records", token, `{"list":"`+list+`","name":"Kaas"}`)
}

func TestTickingOffKeepsAPurchase(t *testing.T) {
	a := newApp(t)
	defer a.Cleanup()
	ada, token := addUser(t, a, "ada@example.com", "owner")
	list := newList(t, a, "Weekend")
	item := send(t, a, http.StatusCreated, http.MethodPost, "/api/app/items", token, `{"list":"`+list+`","name":"Bananen"}`)
	url := "/api/collections/items/records/" + item["id"].(string)
	patch := func(body string) map[string]any {
		t.Helper()
		return send(t, a, http.StatusOK, http.MethodPatch, url, token, body)
	}

	ticked := patch(`{"checked":true}`)
	if ticked["checked"] != true || ticked["checked_by"] != ada.Id || ticked["checked_at"] == "" {
		t.Fatalf("ticked %v", ticked)
	}
	purchases, err := a.FindAllRecords("purchases", dbx.HashExp{"item": item["id"]})
	if err != nil || len(purchases) != 1 || purchases[0].GetString("by") != ada.Id || purchases[0].GetString("key") != "g:bananas" {
		t.Fatalf("purchases %v, %v", purchases, err)
	}

	// putting it back doesn't count as buying it, and ticking it again does
	if unticked := patch(`{"checked":false}`); unticked["checked_by"] != "" {
		t.Errorf("unticked %v", unticked)
	}
	patch(`{"note":"ripe ones"}`)
	patch(`{"checked":true}`)
	if n, _ := a.CountRecords("purchases"); n != 2 {
		t.Errorf("%d purchases", n)
	}

	// nobody writes purchases by hand
	send(t, a, http.StatusForbidden, http.MethodPost, "/api/collections/purchases/records", token,
		`{"list":"`+list+`","name":"Bananen"}`)
}

func TestRenamingToAnItemOnTheList(t *testing.T) {
	a := newApp(t)
	defer a.Cleanup()
	_, token := addUser(t, a, "ada@example.com", "owner")
	list := newList(t, a, "Weekend")
	send(t, a, http.StatusCreated, http.MethodPost, "/api/app/items", token, `{"list":"`+list+`","name":"Kaas"}`)
	brood := send(t, a, http.StatusCreated, http.MethodPost, "/api/app/items", token, `{"list":"`+list+`","name":"Brood"}`)
	url := "/api/collections/items/records/" + brood["id"].(string)

	// "kaas" is the generic cheese, which is already on the list
	send(t, a, http.StatusBadRequest, http.MethodPatch, url, token, `{"name":"kaas","generic":""}`)
	if renamed := send(t, a, http.StatusOK, http.MethodPatch, url, token, `{"name":"Volkoren brood","generic":""}`); renamed["name"] != "Volkoren brood" {
		t.Errorf("renamed %v", renamed)
	}
}
