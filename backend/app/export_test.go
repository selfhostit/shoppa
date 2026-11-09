package app

import (
	"sync"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/selfhostit/shoppa/push"
)

// Sent is a notification a test caught instead of sending.
type Sent struct {
	User string
	Msg  push.Message
}

// CatchPush collects the notifications that would be sent, until the test
// ends. They wait until FlushNotices.
func CatchPush(t testing.TB) func() []Sent {
	var mu sync.Mutex
	var sent []Sent
	wasQuiet, wasMost, wasSend := notifyQuiet, notifyMost, sendPush
	notifyQuiet, notifyMost = time.Hour, time.Hour
	sendPush = func(_ core.App, user string, msg push.Message) {
		mu.Lock()
		defer mu.Unlock()
		sent = append(sent, Sent{user, msg})
	}
	t.Cleanup(func() { notifyQuiet, notifyMost, sendPush = wasQuiet, wasMost, wasSend })
	return func() []Sent {
		mu.Lock()
		defer mu.Unlock()
		return append([]Sent(nil), sent...)
	}
}

// FlushNotices sends the waiting notifications now.
func FlushNotices(a core.App) { notices.flush(a) }

// BringBackStaples runs the morning staples job now.
var BringBackStaples = bringBackStaples
