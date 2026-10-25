package provider

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/yetone/magpie/internal/plugin"
)

// A plugin that leaves out used says nothing of how much of a window is
// spent: a 0 is a reading, absent is not. The window goes as its text,
// never "100% left" (QuotaWindow.Text), and has no share in the allowance
// routing ranks on; an account whose windows all say nothing is unknown to
// routing, not the account with the most left (#3650).
func TestPluginWindowWithoutUsedIsNoReading(t *testing.T) {
	now := time.Now()
	q := quotaOfPlugin(SubscriptionQuota{Provider: "loomy-plugin", User: "u"}, plugin.Usage{Windows: []plugin.UsageWindow{
		{Name: "Credits", Display: "0 / 5000"},
		{Name: "Daily", Used: fptr(0), Display: "0 / 100", Limit: 100},
	}})
	if !q.Windows[0].Text || !q.Windows[0].Aside {
		t.Errorf("not said: Text %v, Aside %v", q.Windows[0].Text, q.Windows[0].Aside)
	}
	if q.Windows[1].Text || q.Windows[1].Aside {
		t.Errorf("a 0 that was said: Text %v, Aside %v", q.Windows[1].Text, q.Windows[1].Aside)
	}
	if a := allowanceOf(q.Windows, now); len(a) != 1 || a[0].Name != "Daily" || a[0].Used != 0 {
		t.Errorf("the allowance routing ranks on: %+v", a)
	}
	only := quotaOfPlugin(SubscriptionQuota{Provider: "loomy-plugin", User: "v"}, plugin.Usage{Windows: []plugin.UsageWindow{
		{Name: "Credits", Display: "0 / 5000"},
	}})
	if a := allowanceOf(only.Windows, now); len(a) != 0 {
		t.Errorf("a window that said nothing is an allowance: %+v", a)
	}
}

// A window that said nothing has no share in the allowance routing ranks
// on, through the decode a plugin's reply takes: the window whose used was
// never sent is absent from it, while one that said 0 is there (#3650).
// This one compiles on the base too, so it is the regression that fails
// before the fix.
func TestAWindowWithoutUsedIsNoAllowance(t *testing.T) {
	var u plugin.Usage
	if err := json.Unmarshal([]byte(`{"windows":[{"name":"Credits","display":"0 / 5000"},{"name":"Daily","used":0,"limit":100,"display":"0 / 100"}]}`), &u); err != nil {
		t.Fatal(err)
	}
	q := quotaOfPlugin(SubscriptionQuota{Provider: "loomy-plugin", User: "u"}, u)
	if a := allowanceOf(q.Windows, time.Now()); len(a) != 1 || a[0].Name != "Daily" {
		t.Fatalf("the allowance: %+v", a)
	}
}
