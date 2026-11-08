package gateway

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/yetone/magpie/internal/proc"
)

func TestModeOnlyAndBrowserMarksMCP(t *testing.T) {
	fresh(t)
	t.Setenv("MAGPIE_ADDR", "")
	t.Setenv("MAGPIE_PUBLIC_URL", "")
	t.Setenv("MAGPIE_TRUST_PROXY", "")
	_, secrets := newCaller(t, "Node")
	f := &fakeHub{on: true, tools: map[string][]HubTool{"": {{Server: "magpie-browser", Tool: "do", Spec: hubSpec("do")}}}, servers: map[string][]HubServer{"": {{Name: "magpie-browser", HereOnly: true, Program: true, Keyed: true}}}}
	withHub(t, f)
	h := lanGuard(New().Handler())
	const there = "192.0.2.7:4000"
	for _, path := range []string{BrowserMCPPath, HubMCPAll} {
		body := toolsList
		if path == HubMCPAll {
			body = toolsCall("magpie-browser__do", `{}`)
		}
		if w := hubPostAt(h, path, there, body, "Sec-Fetch-Mode", "cors"); w.Code != http.StatusUnauthorized {
			t.Errorf("%s unkeyed mode: %d", path, w.Code)
		}
		for _, mark := range [][]string{{"Origin", "http://localhost:5173"}, {"Sec-Fetch-Site", "same-origin"}, {"User-Agent", "Mozilla/5.0"}} {
			headers := append([]string{"Authorization", "Bearer " + secrets[0], "Sec-Fetch-Mode", "cors"}, mark...)
			w := hubPostAt(h, path, there, body, headers...)
			if path == BrowserMCPPath && w.Code != http.StatusForbidden {
				t.Errorf("%s keyed %s: %d", path, mark[0], w.Code)
			}
			if path == HubMCPAll && w.Code == http.StatusOK && strings.Contains(w.Body.String(), `"text":"ran `) {
				t.Errorf("hub ran browser for keyed %s", mark[0])
			}
			if strings.Contains(w.Body.String(), secrets[0]) {
				t.Errorf("%s leaked key", path)
			}
		}
	}
	if got := f.ran(); len(got) != 0 {
		t.Errorf("refused callers ran hub tools: %v", got)
	}
}

func TestNodeFetchKeyedMCP(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node is not installed")
	}
	fresh(t)
	t.Setenv("MAGPIE_ADDR", "")
	t.Setenv("MAGPIE_PUBLIC_URL", "")
	t.Setenv("MAGPIE_TRUST_PROXY", "")
	keys, secrets := newCaller(t, "Node")
	f := &fakeHub{on: true, tools: map[string][]HubTool{"": {{Server: "magpie-browser", Tool: "do", Spec: hubSpec("do")}}}, servers: map[string][]HubServer{"": {{Name: "magpie-browser", HereOnly: true, Program: true, Keyed: true}}}}
	var mu sync.Mutex
	var session string
	f.call = func(ctx context.Context, server, tool string) (json.RawMessage, error) {
		mu.Lock()
		session = HubCallerOf(ctx).Session
		mu.Unlock()
		return json.RawMessage(`{"content":[{"type":"text","text":"ok"}]}`), nil
	}
	withHub(t, f)
	h := lanGuard(New().Handler())
	var fetchMode, fetchSite, fetchOrigin, fetchUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Sec-Fetch-Mode") != "" {
			mu.Lock()
			fetchMode, fetchSite, fetchOrigin, fetchUA = r.Header.Get("Sec-Fetch-Mode"), r.Header.Get("Sec-Fetch-Site"), r.Header.Get("Origin"), r.UserAgent()
			mu.Unlock()
		}
		r.RemoteAddr = "192.0.2.7:4000"
		h.ServeHTTP(w, r)
	}))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := proc.CommandContext(ctx, node, "-e", `
const http = require('node:http');
let input=''; process.stdin.setEncoding('utf8'); process.stdin.on('data', x => input += x);
process.stdin.on('end', async () => {
 const {base,key} = JSON.parse(input);
 const headers = {'authorization':'Bearer '+key,'content-type':'application/json','accept':'application/json, text/event-stream'};
 const body = JSON.stringify({jsonrpc:'2.0',id:1,method:'tools/call',params:{name:'magpie-browser__do',arguments:{}}});
 const init = JSON.stringify({jsonrpc:'2.0',id:1,method:'initialize',params:{protocolVersion:'2025-06-18'}});
 const fetchResult = await fetch(base+'/mcp/magpie/browser',{method:'POST',headers,body:init});
 const fetchBody = await fetchResult.text();
 const control = await new Promise((resolve,reject) => { const req=http.request(base+'/mcp/magpie/browser',{method:'POST',headers},r=>{let b='';r.on('data',x=>b+=x);r.on('end',()=>resolve({status:r.statusCode,body:b}));});req.on('error',reject);req.end(init); });
 const hub = await fetch(base+'/mcp/magpie/hub',{method:'POST',headers,body});
 console.log(JSON.stringify({version:process.version,fetch:fetchResult.status,fetchBody,control,hub:hub.status,hubBody:await hub.text()}));
}).on('error', e => { console.error(e); process.exitCode=1; });`)
	input, err := json.Marshal(map[string]string{"base": srv.URL, "key": secrets[0]})
	if err != nil {
		t.Fatal(err)
	}
	cmd.Stdin = strings.NewReader(string(input))
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("node: %v: %s", err, out)
	}
	var got struct {
		Version   string
		Fetch     int
		FetchBody string
		Control   struct {
			Status int
			Body   string
		}
		Hub     int
		HubBody string
	}
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("node output: %v: %s", err, out)
	}
	t.Logf("runtime %s; fetch=%d http.request=%d hub=%d", got.Version, got.Fetch, got.Control.Status, got.Hub)
	if got.Fetch != http.StatusOK || !strings.Contains(got.FetchBody, "magpie-browser") {
		t.Errorf("Node fetch browser: %d %s", got.Fetch, got.FetchBody)
	}
	if got.Control.Status != http.StatusOK || !strings.Contains(got.Control.Body, "magpie-browser") {
		t.Errorf("Node http.request browser: %d %s", got.Control.Status, got.Control.Body)
	}
	if got.Hub != http.StatusOK || !strings.Contains(got.HubBody, `"text":"ok"`) {
		t.Errorf("Node fetch hub: %d %s", got.Hub, got.HubBody)
	}
	mu.Lock()
	defer mu.Unlock()
	if fetchMode != "cors" || fetchSite != "" || fetchOrigin != "" || strings.HasPrefix(fetchUA, "Mozilla/") {
		t.Errorf("Node fetch headers: mode=%q site=%q origin=%q UA=%q", fetchMode, fetchSite, fetchOrigin, fetchUA)
	}
	if session != "key:"+keys[0].ID+"/" {
		t.Errorf("hub session = %q", session)
	}
}
