package app_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/tools/types"

	"github.com/selfhostit/shoppa/app"
)

func TestStaplesComeBack(t *testing.T) {
	a := newApp(t)
	defer a.Cleanup()
	_, token := addUser(t, a, "ada@example.com", "owner")
	list := newList(t, a, "Weekend")
	item := send(t, a, http.StatusCreated, http.MethodPost, "/api/app/items", token, `{"list":"`+list+`","name":"Toiletpapier"}`)
	itemURL := "/api/collections/items/records/" + item["id"].(string)
	staple := send(t, a, http.StatusOK, http.MethodPost, "/api/collections/staples/records", token,
		`{"item":"`+item["id"].(string)+`","every":14}`)
	next, _ := types.ParseDateTime(staple["next"])
	if days := time.Until(next.Time()).Hours() / 24; days < 13 || days > 14 {
		t.Fatalf("comes back in %.1f days", days)
	}
	send(t, a, http.StatusBadRequest, http.MethodPost, "/api/collections/staples/records", token,
		`{"item":"`+item["id"].(string)+`","every":0}`)

	// bought, and then the day comes
	send(t, a, http.StatusOK, http.MethodPatch, itemURL, token, `{"checked":true}`)
	s, err := a.FindRecordById("staples", staple["id"].(string))
	if err != nil {
		t.Fatal(err)
	}
	s.Set("next", types.NowDateTime().Add(-time.Hour))
	if err := a.SaveNoValidate(s); err != nil {
		t.Fatal(err)
	}
	if err := app.BringBackStaples(a); err != nil {
		t.Fatal(err)
	}
	back, _ := a.FindRecordById("items", item["id"].(string))
	if back.GetBool("checked") {
		t.Error("the staple didn't come back")
	}
	s, _ = a.FindRecordById("staples", staple["id"].(string))
	if !s.GetDateTime("next").Time().After(time.Now().Add(13 * 24 * time.Hour)) {
		t.Errorf("next %v", s.GetDateTime("next"))
	}

	// taking the item off the list stops it
	if err := a.Delete(back); err != nil {
		t.Fatal(err)
	}
	if n, _ := a.CountRecords("staples"); n != 0 {
		t.Errorf("%d staples left", n)
	}
}
