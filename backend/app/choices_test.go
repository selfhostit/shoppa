package app_test

import (
	"net/http"
	"testing"
)

func TestItemChoices(t *testing.T) {
	a := newApp(t)
	defer a.Cleanup()
	_, ada := addUser(t, a, "ada@example.com", "owner")
	_, bob := addUser(t, a, "bob@example.com", "member")
	const url = "/api/collections/item_choices/records"

	// anyone in the household picks a section and an icon for an item
	c := send(t, a, http.StatusOK, http.MethodPost, url, bob, `{"key":"chicken breast small","section":"meat","emoji":"🍗"}`)
	// and anyone changes it
	send(t, a, http.StatusOK, http.MethodPatch, url+"/"+c["id"].(string), ada, `{"emoji":"🥩"}`)
	// one choice for each name
	send(t, a, http.StatusBadRequest, http.MethodPost, url, ada, `{"key":"chicken breast small","section":"deli"}`)
	if n, _ := a.CountRecords("item_choices"); n != 1 {
		t.Errorf("%d choices", n)
	}
	// nobody else sees or makes them
	list := send(t, a, http.StatusOK, http.MethodGet, url, "", "")
	if list["totalItems"].(float64) != 0 {
		t.Errorf("signed out, saw %v", list)
	}
	send(t, a, http.StatusBadRequest, http.MethodPost, url, "", `{"key":"x","section":"meat"}`)
}
