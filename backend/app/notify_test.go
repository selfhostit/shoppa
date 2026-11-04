package app_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"

	"github.com/selfhostit/shoppa/app"
	"github.com/selfhostit/shoppa/push"
)

func named(t *testing.T, a core.App, u *core.Record, name, lang string) {
	t.Helper()
	u.Set("name", name)
	u.Set("lang", lang)
	if err := a.Save(u); err != nil {
		t.Fatal(err)
	}
}

func TestAddedItemsMakeOneNotification(t *testing.T) {
	sent := app.CatchPush(t)
	a := newApp(t)
	defer a.Cleanup()
	anna, token := addUser(t, a, "anna@example.com", "owner")
	ben, _ := addUser(t, a, "ben@example.com", "member")
	named(t, a, anna, "Anna Peeters", "nl")
	named(t, a, ben, "Ben", "fr")
	list := newList(t, a, "Groceries")
	add := func(name string) map[string]any {
		return send(t, a, http.StatusCreated, http.MethodPost, "/api/app/items", token,
			`{"list":"`+list+`","name":"`+name+`"}`)
	}

	add("Bananen")
	add("Brood")
	grass := add("Kattengras")
	// a second time is the same item
	send(t, a, http.StatusOK, http.MethodPost, "/api/app/items", token, `{"list":"`+list+`","name":"brood"}`)
	// and one taken off again before the notification isn't news
	if r, err := a.FindRecordById("items", grass["id"].(string)); err != nil || a.Delete(r) != nil {
		t.Fatal("deleting", err)
	}
	if got := sent(); len(got) != 0 {
		t.Fatalf("sent before the quiet time: %v", got)
	}

	app.FlushNotices(a)
	want := push.Message{
		Title: "Anna added 2 items to Groceries",
		Body:  "Bananes, Pain", // in Ben's language
		URL:   "/?list=" + list,
		Tag:   "list-" + list,
	}
	if got := sent(); len(got) != 1 || got[0].User != ben.Id || got[0].Msg != want {
		t.Fatalf("sent %+v, want one to Ben: %+v", got, want)
	}
}

func TestAPutBackItemIsNewsToo(t *testing.T) {
	sent := app.CatchPush(t)
	a := newApp(t)
	defer a.Cleanup()
	anna, annaToken := addUser(t, a, "anna@example.com", "owner")
	ben, benToken := addUser(t, a, "ben@example.com", "member")
	named(t, a, anna, "Anna", "nl")
	named(t, a, ben, "Ben", "en")
	list := newList(t, a, "Groceries")
	milk := send(t, a, http.StatusCreated, http.MethodPost, "/api/app/items", annaToken,
		`{"list":"`+list+`","name":"Melk"}`)
	url := "/api/collections/items/records/" + milk["id"].(string)
	app.FlushNotices(a)
	send(t, a, http.StatusOK, http.MethodPatch, url, benToken, `{"checked":true}`)

	send(t, a, http.StatusOK, http.MethodPatch, url, benToken, `{"checked":false}`)
	app.FlushNotices(a)
	got := sent()
	if len(got) != 2 || got[0].Msg.Title != "Anna added Milk to Groceries" ||
		got[1].User != anna.Id || got[1].Msg.Title != "Ben added Melk to Groceries" {
		t.Fatalf("sent %+v", got)
	}
}

func TestALongBatchNamesTheFirstFive(t *testing.T) {
	sent := app.CatchPush(t)
	a := newApp(t)
	defer a.Cleanup()
	_, token := addUser(t, a, "anna@example.com", "owner")
	addUser(t, a, "ben@example.com", "member")
	list := newList(t, a, "Groceries")
	for _, name := range []string{"Appels", "Peren", "Kiwi's", "Druiven", "Citroenen", "Limoenen", "Mango"} {
		send(t, a, http.StatusCreated, http.MethodPost, "/api/app/items", token, `{"list":"`+list+`","name":"`+name+`"}`)
	}
	app.FlushNotices(a)
	got := sent()
	if len(got) != 1 || got[0].Msg.Title != "anna added 7 items to Groceries" || !strings.HasSuffix(got[0].Msg.Body, " and 2 more") {
		t.Fatalf("sent %+v", got)
	}
}

func TestNewsOnlyOfWhatIsStillToBuy(t *testing.T) {
	sent := app.CatchPush(t)
	a := newApp(t)
	defer a.Cleanup()
	_, token := addUser(t, a, "anna@example.com", "owner")
	_, benToken := addUser(t, a, "ben@example.com", "member")
	list := newList(t, a, "Groceries")
	milk := send(t, a, http.StatusCreated, http.MethodPost, "/api/app/items", token, `{"list":"`+list+`","name":"Melk"}`)
	// Ben was in the shop and got it already
	send(t, a, http.StatusOK, http.MethodPatch, "/api/collections/items/records/"+milk["id"].(string), benToken, `{"checked":true}`)
	app.FlushNotices(a)
	if got := sent(); len(got) != 0 {
		t.Fatalf("sent %+v", got)
	}
}
