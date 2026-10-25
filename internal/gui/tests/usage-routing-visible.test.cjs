// #3624: a Usage link reveals its routing story even after the Routing page was scrolled.
const assert = require("node:assert/strict");
const fs = require("node:fs/promises");
const path = require("node:path");
const { test } = require("node:test");
const { chromium, webkit } = require("playwright");
const { click, inView } = require("./reader.cjs");

const assets = path.resolve(__dirname, "../assets");
const now = Date.now();

// 130 requests, newest first: a swapped one, one under a dated name, a
// failure (passed on by another computer's magpie), one from before Requested was kept, then plain ones
const ROWS = [{ route_id:123, t:new Date(now-86400e3).toISOString(), agent:"codex", provider:"relay", model:"gpt-6-sol", status:200, ms:50 }];

function server(lang, theme, asked) {
  return async (route) => {
    const req = route.request(), url = new URL(req.url());
    const json = (data) => route.fulfill({ json: data });
    if (url.pathname === "/boot.js") return route.fulfill({ contentType: "text/javascript", body: `window.bootPrefs = {lang:"${lang}",theme:"${theme}",web:false};` });
    if (url.pathname === "/wails/runtime.js") return route.fulfill({ contentType: "text/javascript", body: "export const Window = {};" });
    if (url.pathname === "/api/gateway/trace") {
      await new Promise((r) => setTimeout(r, 100));
      return json({ mine: false, seq: 0, routes: [], totals: { requests: 0, rerouted: 0, errors: 0 }, now: new Date().toISOString() });
    }
    if (url.pathname === "/api/gateway/history") return json({ days: [], routes: [], cut: false });
    if (url.pathname === "/api/gateway/route") {
      if (url.searchParams.get("id") !== "123") return route.fulfill({ status: 404, body: "not found" });
      return json({ id: 123, time: new Date(now - 86400e3).toISOString(), agent: "codex", model: "gpt-6-sol", provider: "relay",
        order: [{ id: "relay", provider: "relay", name: "Relay", model: "gpt-6-sol", kind: "provider", routing: "order" }],
        tries: [{ id: "relay", model: "gpt-6-sol", start: new Date(now - 86400e3).toISOString(), done: true, status: 200, ms: 50 }],
        done: true, status: 200, ms: 50 });
    }
    if (url.pathname === "/api/providers") return json({ providers: [], presets: [], excluded: [], models: [], gateway: { running: false } });
    if (url.pathname === "/api/state") return json({ agents: [], profiles: [], settings: { lang, theme }, fx: { rate: 7.2, at: new Date().toISOString() } });
    if (url.pathname === "/api/usage/requests") {
      asked.push(url.searchParams);
      return json({rows:ROWS,total:1,offset:0,calls:1,agents:[],models:[],callerKeys:[]});
    }
    if (url.pathname === "/api/usage/quotas") return json([]);
    if (url.pathname === "/api/usage") {
      return json({
        calls: 130, errors: 1, input: 160000, output: 14000, cache_read: 69000, cache_write: 2048, reasoning: 300, unpriced: 2, cost: 12.34, bucket: "day",
        series: [{ label: "Mon", input: 80000, output: 7000, calls: 65, cost: 6 }, { label: "Tue", input: 80000, output: 7000, calls: 65, cost: 6.34 }],
        agents: [{ name: "Codex", calls: 65, cost: 6 }], models: [{ name: "gpt-6-sol", calls: 65, cost: 6 }], path: "~/.config/magpie/usage.jsonl",
      });
    }
    if (url.pathname === "/api/sessions") return json({ sessions: [], dirs: [] });
    if (url.pathname === "/api/sessions/stats") return json({ from: "", to: "", days: [], agents: {} });
    if (url.pathname === "/api/groups") return json({ groups: [], models: [] });
    if (url.pathname.startsWith("/api/")) return json({});
    const file = path.join(assets, url.pathname === "/" ? "index.html" : url.pathname);
    const contentType = { ".html": "text/html", ".js": "text/javascript", ".css": "text/css", ".svg": "image/svg+xml", ".png": "image/png" }[path.extname(file)];
    await route.fulfill({ body: await fs.readFile(file), contentType });
  };
}


for (const engine of (process.env.BROWSER ? [process.env.BROWSER] : ["chromium", "webkit"])) {
 for (const lang of ["en", "zh", "zh-TW", "ja", "de"]) {
  test(`${engine} ${lang}: Usage routing navigation reveals the selected story`, async (t) => {
   const browser = await (engine === "webkit" ? webkit.launch() : chromium.launch({channel:"chromium"}));
   t.after(()=>browser.close());
   const tab=await browser.newPage({viewport:{width:1100,height:760},reducedMotion:"reduce"});
   tab.setDefaultTimeout(5000);
   const errors=[]; tab.on("pageerror", e=>errors.push(e.message));
   await tab.route("**/*", server(lang,"light",[]));
   await tab.goto("http://magpie.test/");
   await tab.locator('nav [data-view="usage"], [data-view="usage"]').first().click();
   await tab.locator(":is(#usageTab .opt, .nav-s):visible").nth(1).click();
   await tab.locator(".led tbody tr").first().waitFor();
   await tab.evaluate(async time=>{
    await window.openRoute(123,time);
    // Keep enough history-page space to reproduce a previously scrolled view.
    const space=document.createElement("div"); space.style.cssText="height:3000px;flex:none";
    document.querySelector("#view-routing").append(space);
   },ROWS[0].t);
   const view=tab.locator("#view-routing");
   const box=await view.boundingBox();
   await tab.mouse.move(box.x+box.width-20,box.y+box.height-20); await tab.mouse.wheel(0,1500);

   await tab.waitForFunction(()=>document.querySelector("#view-routing").scrollTop>600);
   await tab.evaluate(()=>window.show("usage"));
   const link=tab.locator(".led tbody tr").first().locator("td").first().locator("button");
   await tab.waitForFunction(()=>document.querySelectorAll(".led tbody tr").length===1);
   await tab.mouse.move(700,600); await tab.mouse.wheel(0,900);
   await link.click();
   await view.waitFor({state:"visible"});
   await tab.waitForFunction(()=>{
    const v=document.querySelector("#view-routing").getBoundingClientRect();
    const h=document.querySelector(".rt-log-head").getBoundingClientRect();
    return h.top>=v.top && h.bottom<=Math.min(v.bottom,innerHeight);
   },null,{timeout:4000});
   assert.match(await tab.locator(".rt-steps").textContent(),/gpt-6-sol/);
   assert.deepEqual(errors,[]);
  });
 }
}
