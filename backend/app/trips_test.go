package app_test

import (
	"net/http"
	"testing"

	"github.com/pocketbase/pocketbase/core"

	"github.com/selfhostit/shoppa/app"
	"github.com/selfhostit/shoppa/push"
)

// notificationsOn saves a device with notifications on for a user.
func notificationsOn(t *testing.T, a core.App, u *core.Record) {
	t.Helper()
	col, err := a.FindCollectionByNameOrId(push.Collection)
	if err != nil {
		t.Fatal(err)
	}
	s := core.NewRecord(col)
	s.Set("user", u.Id)
	s.Set("endpoint", "https://push.example.com/"+u.Id)
	s.Set("p256dh", "key")
	s.Set("auth", "secret")
	if err := a.Save(s); err != nil {
		t.Fatal(err)
	}
}

func TestGoingShoppingTellsTheOthers(t *testing.T) {
	sent := app.CatchPush(t)
	a := newApp(t)
	defer a.Cleanup()
	anna, annaToken := addUser(t, a, "anna@example.com", "owner")
	ben, benToken := addUser(t, a, "ben@example.com", "member")
	addUser(t, a, "cleo@example.com", "member") // no notifications on
	named(t, a, anna, "Anna Peeters", "nl")
	notificationsOn(t, a, anna)
	notificationsOn(t, a, ben)
	list := newList(t, a, "Groceries")
	start := func(body string) map[string]any {
		return send(t, a, http.StatusOK, http.MethodPost, "/api/collections/trips/records", annaToken, body)
	}

	// who is shopping is whoever started it, and it tells whoever can hear it
	first := start(`{"list":"` + list + `","store":" Colruyt ","by":"` + ben.Id + `","notified":["` + anna.Id + `"]}`)
	if first["by"] != anna.Id || first["store"] != "Colruyt" || first["ended"] != "" {
		t.Fatalf("trip %v", first)
	}
	if told, _ := first["notified"].([]any); len(told) != 1 || told[0] != ben.Id {
		t.Fatalf("notified %v, want only Ben", first["notified"])
	}
	want := push.Message{
		Title: "Anna is shopping at Colruyt",
		Body:  "Add what you need to Groceries now.",
		URL:   "/?list=" + list,
		Tag:   "trip-" + list,
	}
	if got := sent(); len(got) != 1 || got[0].User != ben.Id || got[0].Msg != want {
		t.Fatalf("sent %+v, want one to Ben: %+v", got, want)
	}

	// a new trip ends the one before, quietly
	second := start(`{"list":"` + list + `"}`)
	if was, _ := a.FindRecordById("trips", first["id"].(string)); was.GetDateTime("ended").IsZero() {
		t.Error("the first trip didn't end")
	}
	if got := sent(); len(got) != 2 || got[1].Msg.Title != "Anna is shopping" {
		t.Errorf("sent %+v", got)
	}

	// only the shopper ends it, and the people it told hear she's done
	url := "/api/collections/trips/records/" + second["id"].(string)
	send(t, a, http.StatusNotFound, http.MethodPatch, url, benToken, `{"ended":"2026-01-01 00:00:00.000Z"}`)
	ended := send(t, a, http.StatusOK, http.MethodPatch, url, annaToken,
		`{"ended":"2026-01-01 00:00:00.000Z","by":"`+ben.Id+`","notified":[]}`)
	if told, _ := ended["notified"].([]any); ended["by"] != anna.Id || len(told) != 1 {
		t.Errorf("ended %v", ended)
	}
	got := sent()
	if len(got) != 3 || got[2].User != ben.Id || got[2].Msg.Title != "Anna is done shopping" || got[2].Msg.Tag != "trip-"+list {
		t.Fatalf("sent %+v, want Ben told Anna is done", got)
	}

	// ending it again tells nobody
	send(t, a, http.StatusOK, http.MethodPatch, url, annaToken, `{"ended":"2026-01-01 00:00:01.000Z"}`)
	if got := sent(); len(got) != 3 {
		t.Errorf("sent %+v", got)
	}
}

func TestGoingShoppingWithNobodyToTell(t *testing.T) {
	sent := app.CatchPush(t)
	a := newApp(t)
	defer a.Cleanup()
	_, annaToken := addUser(t, a, "anna@example.com", "owner")
	addUser(t, a, "ben@example.com", "member")
	list := newList(t, a, "Groceries")

	trip := send(t, a, http.StatusOK, http.MethodPost, "/api/collections/trips/records", annaToken, `{"list":"`+list+`"}`)
	if told, _ := trip["notified"].([]any); len(told) != 0 {
		t.Errorf("notified %v", trip["notified"])
	}
	if got := sent(); len(got) != 0 {
		t.Errorf("sent %+v", got)
	}
}
