package app

import (
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"

	"github.com/selfhostit/shoppa/jobs"
)

// staplesJob puts staples that are due back on their lists, every morning.
const staplesJob = "staples"

func registerStaples(app core.App) {
	// a staple first comes back a full period after it is set, or changed
	setNext := func(e *core.RecordEvent) error {
		if e.Record.IsNew() || e.Record.GetInt("every") != e.Record.Original().GetInt("every") {
			e.Record.Set("next", nextStaple(time.Now(), e.Record.GetInt("every")))
		}
		return e.Next()
	}
	app.OnRecordCreate("staples").BindFunc(setNext)
	app.OnRecordUpdate("staples").BindFunc(setNext)
	jobs.MustAdd(app, staplesJob, "0 6 * * *", bringBackStaples)
}

// nextStaple is the day a staple comes back, every days after from, at the
// start of that day so the morning job picks it up.
func nextStaple(from time.Time, every int) types.DateTime {
	y, m, d := from.AddDate(0, 0, max(every, 1)).Date()
	next, _ := types.ParseDateTime(time.Date(y, m, d, 0, 0, 0, 0, from.Location()))
	return next
}

// bringBackStaples puts each staple that is due back on its list, unless it
// is still on it, and sets when it comes back next.
func bringBackStaples(app core.App) error {
	now := time.Now()
	due, err := app.FindAllRecords("staples", dbx.NewExp("next <= {:now}", dbx.Params{"now": types.NowDateTime().String()}))
	if err != nil {
		return err
	}
	for _, s := range due {
		item, err := app.FindRecordById("items", s.GetString("item"))
		if err != nil {
			continue // the item went, and its staple with it
		}
		if item.GetBool("checked") {
			item.Set("checked", false)
			item.Set("checked_by", "")
			item.Set("added_by", "")
			if err := app.Save(item); err != nil {
				app.Logger().Error("bringing back a staple", "item", item.Id, "error", err)
				continue
			}
		}
		s.Set("next", nextStaple(now, s.GetInt("every")))
		if err := app.SaveNoValidate(s); err != nil {
			return err
		}
	}
	return nil
}
