package app

import (
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"github.com/selfhostit/shoppa/products"
	"github.com/selfhostit/shoppa/push"
)

// "Anna added 3 items to Groceries": the items a person puts on a list are
// collected and sent as one notification to everyone else, once they've been
// quiet for two minutes, or after five at most. Each list has its own tag, so
// a newer notification replaces the older one on the phone. A batch waiting
// when the server stops is lost, which only costs a notification.
var (
	notifyQuiet = 2 * time.Minute
	notifyMost  = 5 * time.Minute
	// sendPush is push.ToUser; tests catch the messages instead
	sendPush = push.ToUser
)

// maxNamed is how many item names a notification lists before "and 2 more".
const maxNamed = 5

type batchKey struct{ list, person string }

type batch struct {
	app   core.App
	items []string // item ids, in the order they were added
	first time.Time
	timer *time.Timer
}

type notifier struct {
	mu      sync.Mutex
	batches map[batchKey]*batch
}

var notices = &notifier{batches: map[batchKey]*batch{}}

// added notes that a person put an item on a list.
func (n *notifier) added(app core.App, item *core.Record, person string) {
	if person == "" {
		return
	}
	k := batchKey{item.GetString("list"), person}
	n.mu.Lock()
	defer n.mu.Unlock()
	b := n.batches[k]
	if b == nil {
		b = &batch{app: app, first: time.Now()}
		n.batches[k] = b
	} else {
		b.timer.Stop()
	}
	if !slices.Contains(b.items, item.Id) {
		b.items = append(b.items, item.Id)
	}
	b.timer = time.AfterFunc(waitFor(b.first, time.Now()), func() { n.send(app, k) })
}

// waitFor is how long to wait for more items before sending: the quiet time,
// but never past the longest wait since the first item.
func waitFor(first, now time.Time) time.Duration {
	return max(min(notifyQuiet, first.Add(notifyMost).Sub(now)), 0)
}

// forget drops an app's batches, when it stops.
func (n *notifier) forget(app core.App) {
	n.mu.Lock()
	defer n.mu.Unlock()
	for k, b := range n.batches {
		if b.app.DataDir() == app.DataDir() {
			b.timer.Stop()
			delete(n.batches, k)
		}
	}
}

// flush sends an app's batches now.
func (n *notifier) flush(app core.App) {
	n.mu.Lock()
	var keys []batchKey
	for k, b := range n.batches {
		if b.app.DataDir() != app.DataDir() {
			continue
		}
		b.timer.Stop()
		keys = append(keys, k)
	}
	n.mu.Unlock()
	for _, k := range keys {
		n.send(app, k)
	}
}

func (n *notifier) send(app core.App, k batchKey) {
	n.mu.Lock()
	b := n.batches[k]
	delete(n.batches, k)
	n.mu.Unlock()
	if b == nil {
		return
	}
	if err := notifyAdded(app, k.list, k.person, b.items); err != nil {
		app.Logger().Warn("Could not send the added items notification", "list", k.list, "error", err)
	}
}

// notifyAdded tells everyone but the person which of these items are still
// on the list, each in their own language.
func notifyAdded(app core.App, listID, personID string, itemIDs []string) error {
	list, err := app.FindRecordById("lists", listID)
	if err != nil {
		return nil // the list is gone, and so is the news
	}
	person, err := app.FindRecordById("users", personID)
	if err != nil {
		return err
	}
	ids := make([]any, len(itemIDs))
	for i, id := range itemIDs {
		ids[i] = id
	}
	items, err := app.FindAllRecords("items", dbx.In("id", ids...), dbx.HashExp{"checked": false})
	if err != nil || len(items) == 0 {
		return err
	}
	// in the order they were added
	slices.SortFunc(items, func(a, b *core.Record) int {
		return slices.Index(itemIDs, a.Id) - slices.Index(itemIDs, b.Id)
	})

	others, err := app.FindAllRecords("users", dbx.Not(dbx.HashExp{"id": personID}))
	if err != nil {
		return err
	}
	for _, u := range others {
		lang := personLang(u)
		names := make([]string, len(items))
		for i, it := range items {
			names[i] = itemName(app, it, lang)
		}
		sendPush(app, u.Id, addedMessage(firstName(person), list, names))
	}
	return nil
}

// addedMessage is "Anna added Melk to Groceries", or for more than one item
// "Anna added 3 items to Groceries" with their names below.
func addedMessage(who string, list *core.Record, names []string) push.Message {
	msg := push.Message{
		URL: "/?list=" + list.Id,
		Tag: "list-" + list.Id,
	}
	where := list.GetString("name")
	if len(names) == 1 {
		msg.Title = fmt.Sprintf("%s added %s to %s", who, names[0], where)
		return msg
	}
	msg.Title = fmt.Sprintf("%s added %d items to %s", who, len(names), where)
	if len(names) > maxNamed {
		msg.Body = fmt.Sprintf("%s and %d more", strings.Join(names[:maxNamed], ", "), len(names)-maxNamed)
	} else {
		msg.Body = strings.Join(names, ", ")
	}
	return msg
}

// itemName is an item's name in a language: its product's or generic's, or
// what was typed.
func itemName(app core.App, item *core.Record, lang string) string {
	if id := item.GetString("product"); id != "" {
		if p, err := app.FindRecordById("products", id); err == nil {
			return products.Name(p, lang)
		}
	}
	if id := item.GetString("generic"); id != "" {
		if g, err := app.FindRecordById("generics", id); err == nil {
			return genericName(g, lang)
		}
	}
	return item.GetString("name")
}

// firstName is "Anna" for "Anna Peeters", or the start of the email.
func firstName(u *core.Record) string {
	name := strings.TrimSpace(u.GetString("name"))
	if name == "" {
		name, _, _ = strings.Cut(u.GetString("email"), "@")
	}
	first, _, _ := strings.Cut(name, " ")
	return first
}
