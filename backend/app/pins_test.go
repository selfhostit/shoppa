package app_test

import (
	"net/http"
	"testing"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/tests"
)

func TestPinningAProduct(t *testing.T) {
	a := newApp(t)
	defer a.Cleanup()
	_, ada := addUser(t, a, "ada@example.com", "owner")
	bob, bobToken := addUser(t, a, "bob@example.com", "member")
	withLang(t, a, bob, "fr")
	list := newList(t, a, "Weekend")
	boni := newProduct(t, a, "5410000000001", "Halfvolle melk", "Lait demi-écrémé")
	add := func(want int, token, body string) map[string]any {
		t.Helper()
		return send(t, a, want, http.MethodPost, "/api/app/items", token, body)
	}
	pin := func(want int, token, item, product string) map[string]any {
		t.Helper()
		return send(t, a, want, http.MethodPost, "/api/app/items/"+item+"/pin", token, `{"product":"`+product+`"}`)
	}
	pins := func() (int, int) {
		t.Helper()
		all, err := a.FindAllRecords("pins", dbx.HashExp{"product": boni.Id})
		if err != nil || len(all) > 1 {
			t.Fatalf("pins %v, %v", all, err)
		}
		if len(all) == 0 {
			return 0, 0
		}
		return len(all), all[0].GetInt("uses")
	}

	// "melk" becomes the Boni milk, named in the language of whoever picked it
	melk := add(http.StatusCreated, ada, `{"list":"`+list+`","name":"melk","quantity":"2×"}`)
	generic := melk["generic"].(string)
	picked := pin(http.StatusOK, bobToken, melk["id"].(string), boni.Id)
	if picked["id"] != melk["id"] || picked["product"] != boni.Id || picked["generic"] != "" ||
		picked["name"] != "Lait demi-écrémé" || picked["quantity"] != "2×" {
		t.Fatalf("picked %v", picked)
	}
	if n, uses := pins(); n != 1 || uses != 1 {
		t.Fatalf("%d pins, %d uses", n, uses)
	}
	if p, _ := a.FindFirstRecordByData("pins", "product", boni.Id); p.GetString("generic") != generic {
		t.Errorf("pinned to %s, not %s", p.GetString("generic"), generic)
	}

	// adding the pinned product again counts as a use
	send(t, a, http.StatusOK, http.MethodPatch, "/api/collections/items/records/"+melk["id"].(string), ada, `{"checked":true}`)
	if back := add(http.StatusOK, ada, `{"list":"`+list+`","product":"`+boni.Id+`"}`); back["id"] != melk["id"] || back["checked"] != false {
		t.Fatalf("back %v", back)
	}
	if _, uses := pins(); uses != 2 {
		t.Errorf("%d uses", uses)
	}

	// "melk" again while the Boni milk is on the list: pinning it makes them one
	again := add(http.StatusCreated, ada, `{"list":"`+list+`","name":"melk","note":"for the coffee"}`)
	merged := pin(http.StatusOK, ada, again["id"].(string), boni.Id)
	if merged["id"] != melk["id"] || merged["note"] != "for the coffee" || merged["quantity"] != "2×" {
		t.Fatalf("merged %v", merged)
	}
	if _, err := a.FindRecordById("items", again["id"].(string)); err == nil {
		t.Error("the generic item is still there")
	}
	if _, uses := pins(); uses != 3 {
		t.Errorf("%d uses", uses)
	}

	// only a generic item can be pinned
	pin(http.StatusBadRequest, ada, melk["id"].(string), boni.Id)
	free := add(http.StatusCreated, ada, `{"list":"`+list+`","name":"Kattengras"}`)
	pin(http.StatusBadRequest, ada, free["id"].(string), boni.Id)
	pin(http.StatusNotFound, ada, "nosuchitem12345", boni.Id)
	pin(http.StatusNotFound, ada, free["id"].(string), "nosuchproduct12")

	// pins are made by picking, not by hand, and anyone can unpin
	send(t, a, http.StatusForbidden, http.MethodPost, "/api/collections/pins/records", ada,
		`{"generic":"`+generic+`","product":"`+boni.Id+`"}`)
	p, _ := a.FindFirstRecordByData("pins", "product", boni.Id)
	unpin := tests.ApiScenario{
		Method:                http.MethodDelete,
		URL:                   "/api/collections/pins/records/" + p.Id,
		Headers:               map[string]string{"Authorization": bobToken},
		ExpectedStatus:        http.StatusNoContent,
		TestAppFactory:        func(testing.TB) *tests.TestApp { return a },
		DisableTestAppCleanup: true,
	}
	unpin.Test(t)
	if n, _ := pins(); n != 0 {
		t.Error("still pinned")
	}
}
