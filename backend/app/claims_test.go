package app_test

import (
	"net/http"
	"testing"
)

func TestClaimingAnItem(t *testing.T) {
	a := newApp(t)
	defer a.Cleanup()
	ada, adaToken := addUser(t, a, "ada@example.com", "owner")
	bob, bobToken := addUser(t, a, "bob@example.com", "member")
	list := newList(t, a, "Weekend")
	item := send(t, a, http.StatusCreated, http.MethodPost, "/api/app/items", adaToken, `{"list":"`+list+`","name":"Taart"}`)
	url := "/api/collections/items/records/" + item["id"].(string)
	patch := func(token, body string) map[string]any {
		t.Helper()
		return send(t, a, http.StatusOK, http.MethodPatch, url, token, body)
	}

	// you claim it for yourself, whoever you name
	if got := patch(bobToken, `{"claimed_by":"`+ada.Id+`"}`); got["claimed_by"] != bob.Id {
		t.Fatalf("claimed %v", got)
	}
	// other changes leave the claim alone
	if got := patch(adaToken, `{"note":"with cream"}`); got["claimed_by"] != bob.Id {
		t.Errorf("after a note %v", got)
	}
	// anyone lets it go
	if got := patch(adaToken, `{"claimed_by":""}`); got["claimed_by"] != "" {
		t.Errorf("let go %v", got)
	}
	// bought, it is nobody's
	patch(bobToken, `{"claimed_by":"`+bob.Id+`"}`)
	if got := patch(bobToken, `{"checked":true}`); got["claimed_by"] != "" {
		t.Errorf("bought %v", got)
	}
}
