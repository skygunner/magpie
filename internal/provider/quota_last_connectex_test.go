package provider

import (
	"errors"
	"net/url"
	"testing"
	"time"
)

// #5341: Windows' refused connection is transient too; a bad sign-in is not.
func TestKeepLastAfterWindowsConnectionRefused(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	reset := func() {
		lastQuotas.Lock()
		lastQuotas.m, lastQuotas.loaded = nil, false
		lastQuotas.Unlock()
	}
	reset()
	t.Cleanup(reset)
	later := time.Now().Add(time.Hour)
	good := SubscriptionQuota{Provider: "codex", User: "windows-test", Windows: []QuotaWindow{
		{Name: "5 hours", Used: 40, ResetsAt: &later, Span: 5 * time.Hour},
	}}
	keepLast(good, "")
	err := codexUsageWords(&url.Error{Op: "Get", URL: "http://127.0.0.1:1/backend-api/wham/usage",
		Err: errors.New("dial tcp 127.0.0.1:1: connectex: No connection could be made because the target machine actively refused it.")})
	q := keepLast(SubscriptionQuota{Provider: good.Provider, User: good.User, Error: err.Error()}, "")
	if q.Error != "" || q.AsOf == nil || len(q.Windows) != 1 || q.Windows[0].Used != 40 {
		t.Fatalf("connection failure lost the last reading: %+v", q)
	}
	denied := SubscriptionQuota{Provider: good.Provider, User: good.User, Error: "ChatGPT answered the usage read with HTTP 401 Unauthorized"}
	if q := keepLast(denied, ""); q.Error != denied.Error || q.AsOf != nil {
		t.Fatalf("bad sign-in hidden by the last reading: %+v", q)
	}
}
