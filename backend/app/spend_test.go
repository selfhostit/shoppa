package app_test

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/tests"

	"github.com/selfhostit/shoppa/app"
)

func TestSpendingAddsUpPrices(t *testing.T) {
	a := newApp(t)
	defer a.Cleanup()
	_, token := addUser(t, a, "ada@example.com", "owner")
	groceries := newList(t, a, "Groceries")
	camping := newList(t, a, "Camping")
	buy := func(list, name, body string) string {
		t.Helper()
		item := send(t, a, http.StatusCreated, http.MethodPost, "/api/app/items", token, `{"list":"`+list+`","name":"`+name+`"}`)
		url := "/api/collections/items/records/" + item["id"].(string)
		send(t, a, http.StatusOK, http.MethodPatch, url, token, body)
		return url
	}

	// priced when ticked, or afterwards from the receipt
	buy(groceries, "Melk", `{"price":1.5,"checked":true}`)
	bread := buy(groceries, "Brood", `{"checked":true}`)
	send(t, a, http.StatusOK, http.MethodPatch, bread, token, `{"price":3.2}`)
	buy(camping, "Gas", `{"price":7,"checked":true}`)
	buy(camping, "Tent pegs", `{"checked":true}`) // never priced

	var got []app.MonthSpend
	(&tests.ApiScenario{
		Method:          http.MethodGet,
		URL:             "/api/app/spend",
		Headers:         map[string]string{"Authorization": token},
		ExpectedStatus:  http.StatusOK,
		ExpectedContent: []string{"["},
		TestAppFactory:  func(testing.TB) *tests.TestApp { return a },
		AfterTestFunc: func(t testing.TB, _ *tests.TestApp, res *http.Response) {
			data, _ := io.ReadAll(res.Body)
			_ = json.Unmarshal(data, &got)
		},
		DisableTestAppCleanup: true,
	}).Test(t)

	month := time.Now().Format("2006-01")
	if len(got) != 1 || got[0].Month != month || got[0].Total < 11.69 || got[0].Total > 11.71 || len(got[0].Lists) != 2 {
		t.Fatalf("spend %+v", got)
	}
	if l := got[0].Lists[0]; l.Name != "Camping" || l.Total != 7 || l.Priced != 1 {
		t.Errorf("first list %+v", l)
	}
	if l := got[0].Lists[1]; l.Name != "Groceries" || l.Priced != 2 {
		t.Errorf("second list %+v", l)
	}

	(&tests.ApiScenario{
		Method: http.MethodGet, URL: "/api/app/spend", Body: strings.NewReader(""),
		ExpectedStatus: http.StatusUnauthorized, ExpectedContent: []string{"{"},
		TestAppFactory:        func(testing.TB) *tests.TestApp { return a },
		DisableTestAppCleanup: true,
	}).Test(t)
}
