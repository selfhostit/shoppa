package app_test

import (
	"net/http"
	"testing"

	"github.com/pocketbase/dbx"
)

func TestCopyingAList(t *testing.T) {
	a := newApp(t)
	defer a.Cleanup()
	ada, token := addUser(t, a, "ada@example.com", "owner")
	_, bobToken := addUser(t, a, "bob@example.com", "member")
	trip := send(t, a, http.StatusOK, http.MethodPost, "/api/collections/lists/records", token,
		`{"name":"Ardennes","kind":"camping"}`)
	from := trip["id"].(string)
	add := func(body string) map[string]any {
		return send(t, a, http.StatusCreated, http.MethodPost, "/api/app/items", bobToken, `{"list":"`+from+`",`+body+`}`)
	}
	add(`"name":"Tent pegs","quantity":"20×","note":"the long ones"`)
	ticked := add(`"name":"Gas cartridge"`)
	send(t, a, http.StatusOK, http.MethodPatch, "/api/collections/items/records/"+ticked["id"].(string), bobToken, `{"checked":true}`)

	copied := send(t, a, http.StatusCreated, http.MethodPost, "/api/app/lists/"+from+"/copy", token, `{"name":"  Ardennes   again "}`)
	if copied["name"] != "Ardennes again" || copied["kind"] != "camping" {
		t.Fatalf("copied %v", copied)
	}
	items, err := a.FindAllRecords("items", dbx.HashExp{"list": copied["id"]})
	if err != nil || len(items) != 2 {
		t.Fatalf("items %v, %v", items, err)
	}
	for _, i := range items {
		if i.GetBool("checked") || i.GetString("added_by") != ada.Id {
			t.Errorf("copied item %v", i)
		}
		if i.GetString("name") == "Tent pegs" && (i.GetString("quantity") != "20×" || i.GetString("note") != "the long ones") {
			t.Errorf("tent pegs %v", i)
		}
	}
	// the old list stays as it was
	if n, _ := a.CountRecords("items", dbx.HashExp{"list": from}); n != 2 {
		t.Errorf("%d items left on the old list", n)
	}

	send(t, a, http.StatusBadRequest, http.MethodPost, "/api/app/lists/"+from+"/copy", token, `{"name":" "}`)
	send(t, a, http.StatusNotFound, http.MethodPost, "/api/app/lists/nope/copy", token, `{"name":"x"}`)
	send(t, a, http.StatusUnauthorized, http.MethodPost, "/api/app/lists/"+from+"/copy", "", `{"name":"x"}`)
}
