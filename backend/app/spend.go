package app

import (
	"net/http"
	"strconv"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

// spendMonths is how many months the spending overview goes back.
const spendMonths = 6

func registerSpend(app core.App) {
	// a price entered after the shop, on a bought item, goes on its purchase
	app.OnRecordAfterUpdateSuccess("items").BindFunc(func(e *core.RecordEvent) error {
		if e.Record.GetBool("checked") && e.Record.Original().GetBool("checked") &&
			e.Record.GetFloat("price") != e.Record.Original().GetFloat("price") {
			if err := repriceLastPurchase(e.App, e.Record); err != nil {
				e.App.Logger().Error("pricing a purchase", "item", e.Record.Id, "error", err)
			}
		}
		return e.Next()
	})
}

func registerSpendRoutes(se *core.ServeEvent) {
	se.Router.GET("/api/app/spend", spend).Bind(apis.RequireAuth())
}

// repriceLastPurchase sets the price of an item's latest purchase to the
// item's.
func repriceLastPurchase(app core.App, item *core.Record) error {
	last, err := app.FindRecordsByFilter("purchases", "item = {:item}", "-created", 1, 0, dbx.Params{"item": item.Id})
	if err != nil || len(last) == 0 {
		return err
	}
	last[0].Set("price", item.GetFloat("price"))
	return app.Save(last[0])
}

// ListSpend is what one list cost in a month.
type ListSpend struct {
	List  string  `json:"list" db:"list"`
	Name  string  `json:"name" db:"name"`
	Total float64 `json:"total" db:"total"`
	// how many purchases had a price
	Priced int `json:"priced" db:"priced"`
}

// MonthSpend is what the household spent in a month, newest first.
type MonthSpend struct {
	Month string      `json:"month"` // "2026-11"
	Total float64     `json:"total"`
	Lists []ListSpend `json:"lists"`
}

// spend adds up the prices of what was bought, per month and list, for the
// last spendMonths months, in the server's time zone.
func spend(e *core.RequestEvent) error {
	var rows []struct {
		Month string `db:"month"`
		ListSpend
	}
	err := e.App.DB().NewQuery(`
		SELECT strftime('%Y-%m', p.created, 'localtime') AS month, p.list AS list,
			COALESCE(l.name, '') AS name, SUM(p.price) AS total, COUNT(*) AS priced
		FROM purchases p LEFT JOIN lists l ON l.id = p.list
		WHERE p.price > 0 AND p.created >= datetime('now', 'localtime', 'start of month', {:back}, 'utc')
		GROUP BY month, p.list
		ORDER BY month DESC, total DESC`).
		Bind(dbx.Params{"back": "-" + strconv.Itoa(spendMonths-1) + " months"}).
		All(&rows)
	if err != nil {
		return err
	}
	months := []MonthSpend{}
	for _, r := range rows {
		if len(months) == 0 || months[len(months)-1].Month != r.Month {
			months = append(months, MonthSpend{Month: r.Month, Lists: []ListSpend{}})
		}
		m := &months[len(months)-1]
		m.Total += r.Total
		m.Lists = append(m.Lists, r.ListSpend)
	}
	return e.JSON(http.StatusOK, months)
}
