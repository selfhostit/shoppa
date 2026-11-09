package app_test

import (
	"net/http"
	"testing"

	"github.com/selfhostit/shoppa/app"
	"github.com/selfhostit/shoppa/push"
)

func TestGoingShoppingTellsTheOthers(t *testing.T) {
	sent := app.CatchPush(t)
	a := newApp(t)
	defer a.Cleanup()
	anna, annaToken := addUser(t, a, "anna@example.com", "owner")
	ben, benToken := addUser(t, a, "ben@example.com", "member")
	named(t, a, anna, "Anna Peeters", "nl")
	list := newList(t, a, "Groceries")
	start := func(body string) map[string]any {
		return send(t, a, http.StatusOK, http.MethodPost, "/api/collections/trips/records", annaToken, body)
	}

	// who is shopping is whoever started it
	first := start(`{"list":"` + list + `","store":" Colruyt ","by":"` + ben.Id + `"}`)
	if first["by"] != anna.Id || first["store"] != "Colruyt" || first["ended"] != "" {
		t.Fatalf("trip %v", first)
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

	// a new trip ends the one before
	start(`{"list":"` + list + `"}`)
	if was, _ := a.FindRecordById("trips", first["id"].(string)); was.GetDateTime("ended").IsZero() {
		t.Error("the first trip didn't end")
	}
	if got := sent(); len(got) != 2 || got[1].Msg.Title != "Anna is shopping" {
		t.Errorf("sent %+v", got)
	}

	// only the shopper ends it
	url := "/api/collections/trips/records/" + first["id"].(string)
	send(t, a, http.StatusNotFound, http.MethodPatch, url, benToken, `{"ended":"2026-01-01 00:00:00.000Z"}`)
	if ended := send(t, a, http.StatusOK, http.MethodPatch, url, annaToken, `{"ended":"2026-01-01 00:00:00.000Z","by":"`+ben.Id+`"}`); ended["by"] != anna.Id {
		t.Errorf("ended %v", ended)
	}
}
