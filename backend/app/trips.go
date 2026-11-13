package app

import (
	"fmt"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"

	"github.com/selfhostit/shoppa/push"
)

// Trips: "Anna is shopping at Colruyt". Starting one tells everyone else with
// notifications on to add what they need now, and the trip keeps who that was
// (notified), so the shopper sees who got the news. Ending it tells the same
// people she's done, replacing the first notification.
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
		told, err := reachableOthers(e.App, by)
		if err != nil {
			return err
		}
		e.Record.Set("notified", told)
		if err := endTrips(e.App, by); err != nil {
			return err
		}
		return e.Next()
	})
	app.OnRecordAfterCreateSuccess("trips").BindFunc(func(e *core.RecordEvent) error {
		if err := notifyTrip(e.App, e.Record, tripMessage); err != nil {
			e.App.Logger().Warn("Could not send the trip notification", "trip", e.Record.Id, "error", err)
		}
		return e.Next()
	})
	app.OnRecordUpdateRequest("trips").BindFunc(func(e *core.RecordRequestEvent) error {
		// a trip only ends; it doesn't move to another list or person, and
		// whom it told stays as it was
		was := e.Record.Original()
		for _, f := range []string{"by", "list", "notified"} {
			e.Record.Set(f, was.Get(f))
		}
		ending := was.GetDateTime("ended").IsZero() && !e.Record.GetDateTime("ended").IsZero()
		if err := e.Next(); err != nil {
			return err
		}
		if ending {
			if err := notifyTrip(e.App, e.Record, doneMessage); err != nil {
				e.App.Logger().Warn("Could not send the trip done notification", "trip", e.Record.Id, "error", err)
			}
		}
		return nil
	})
}

// endTrips ends a person's trips that are still going, quietly: a new trip's
// notification replaces theirs.
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

// reachableOthers is everyone but the shopper who has notifications on.
func reachableOthers(app core.App, shopper string) ([]string, error) {
	others, err := app.FindAllRecords("users", dbx.Not(dbx.HashExp{"id": shopper}))
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(others))
	for i, u := range others {
		ids[i] = u.Id
	}
	return push.Reachable(app, ids)
}

// notifyTrip sends the people a trip told a message about it.
func notifyTrip(app core.App, trip *core.Record, message func(who string, list *core.Record, store string) push.Message) error {
	list, err := app.FindRecordById("lists", trip.GetString("list"))
	if err != nil {
		return nil // the list is gone, and so is the news
	}
	shopper, err := app.FindRecordById("users", trip.GetString("by"))
	if err != nil {
		return err
	}
	msg := message(firstName(shopper), list, trip.GetString("store"))
	for _, id := range trip.GetStringSlice("notified") {
		sendPush(app, id, msg)
	}
	return nil
}

// tripMessage is "Anna is shopping at Colruyt" with "Add what you need to
// Groceries now." below.
func tripMessage(who string, list *core.Record, store string) push.Message {
	return push.Message{
		Title: who + " is shopping" + at(store),
		Body:  fmt.Sprintf("Add what you need to %s now.", list.GetString("name")),
		URL:   "/?list=" + list.Id,
		Tag:   tripTag(list),
	}
}

// doneMessage is "Anna is done shopping at Colruyt", in place of the
// notification that she started.
func doneMessage(who string, list *core.Record, store string) push.Message {
	return push.Message{
		Title: who + " is done shopping" + at(store),
		Body:  fmt.Sprintf("Anything new on %s waits for the next trip.", list.GetString("name")),
		URL:   "/?list=" + list.Id,
		Tag:   tripTag(list),
	}
}

func at(store string) string {
	if store == "" {
		return ""
	}
	return " at " + store
}

// tripTag makes a trip's notifications replace each other on a phone.
func tripTag(list *core.Record) string { return "trip-" + list.Id }
