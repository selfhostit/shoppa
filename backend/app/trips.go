package app

import (
	"fmt"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"

	"github.com/selfhostit/shoppa/push"
)

func registerTrips(app core.App) {
	// the shopper is whoever started it, and their earlier trip is over
	app.OnRecordCreateRequest("trips").BindFunc(func(e *core.RecordRequestEvent) error {
		by := personID(e.Auth)
		if by == "" {
			return e.ForbiddenError("Only people go shopping.", nil)
		}
		e.Record.Set("by", by)
		e.Record.Set("ended", "")
		e.Record.Set("store", strings.TrimSpace(e.Record.GetString("store")))
		if err := endTrips(e.App, by); err != nil {
			return err
		}
		return e.Next()
	})
	app.OnRecordUpdateRequest("trips").BindFunc(func(e *core.RecordRequestEvent) error {
		// a trip only ends; it doesn't move to another list or person
		e.Record.Set("by", e.Record.Original().GetString("by"))
		e.Record.Set("list", e.Record.Original().GetString("list"))
		return e.Next()
	})
	app.OnRecordAfterCreateSuccess("trips").BindFunc(func(e *core.RecordEvent) error {
		if err := notifyTrip(e.App, e.Record); err != nil {
			e.App.Logger().Warn("Could not send the trip notification", "trip", e.Record.Id, "error", err)
		}
		return e.Next()
	})
}

// endTrips ends a person's trips that are still going.
func endTrips(app core.App, person string) error {
	open, err := app.FindAllRecords("trips", dbx.HashExp{"by": person, "ended": ""})
	if err != nil {
		return err
	}
	for _, t := range open {
		t.Set("ended", types.NowDateTime())
		if err := app.Save(t); err != nil {
			return err
		}
	}
	return nil
}

// notifyTrip tells everyone but the shopper to add what they need now.
func notifyTrip(app core.App, trip *core.Record) error {
	list, err := app.FindRecordById("lists", trip.GetString("list"))
	if err != nil {
		return nil
	}
	shopper, err := app.FindRecordById("users", trip.GetString("by"))
	if err != nil {
		return err
	}
	others, err := app.FindAllRecords("users", dbx.Not(dbx.HashExp{"id": shopper.Id}))
	if err != nil {
		return err
	}
	msg := tripMessage(firstName(shopper), list, trip.GetString("store"))
	for _, u := range others {
		sendPush(app, u.Id, msg)
	}
	return nil
}

// tripMessage is "Anna is shopping at Colruyt" with "Add what you need to
// Groceries now." below.
func tripMessage(who string, list *core.Record, store string) push.Message {
	title := who + " is shopping"
	if store != "" {
		title += " at " + store
	}
	return push.Message{
		Title: title,
		Body:  fmt.Sprintf("Add what you need to %s now.", list.GetString("name")),
		URL:   "/?list=" + list.Id,
		Tag:   "trip-" + list.Id,
	}
}
