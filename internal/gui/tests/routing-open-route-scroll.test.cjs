// Run with Node's test runner and Playwright on the module path; see README.md.
// #3624: clicking View routing / 查看路由 on a historical Usage row opens the
// Routing page on that request, but the page keeps the scroll it had from the
// reader's earlier visit, and the picked request's story (.rt-log) — above the
// request list — stays out of sight. Opening from another page is a click that
// asks to go somewhere, so the story comes into view; a pick inside Routing
// keeps the page where it is. Chromium and WebKit, English and Chinese.
const assert = require("node:assert/strict");
const fs = require("node:fs/promises");
const path = require("node:path");
const { test } = require("node:test");
const { chromium, webkit } = require("playwright");

const assets = path.resolve(__dirname, "../assets");
const now = new Date();
const day = [now.getFullYear(), now.getMonth() + 1, now.getDate()].map((n) => String(n).padStart(2, "0")).join("-");
// a time as the gateway writes it: local, with its offset, so its date is
// the local day its history keeps it under
const stamp = (d) => {
  const o = -d.getTimezoneOffset(), p = (n) => String(Math.floor(Math.abs(n))).padStart(2, "0");
  return new Date(d.getTime() + o * 60e3).toISOString().slice(0, 19) + (o < 0 ? "-" : "+") + p(o / 60) + ":" + p(o % 60);
};
function req(id) {
  const time = stamp(new Date(now.getTime() - (60 - id) * 60e3));
  const seat = { id: "relay", provider: "relay", name: "Relay", who: "Test account", kind: "account", model: "model-b" };
  return { id, seq: id, time, agent: "codex", session: "chat-" + id, model: "relay/model-b", provider: "relay",
    order: [seat], tries: [{ id: seat.id, model: seat.model, start: time, done: true, status: 200, ms: 4500 }],
    done: true, status: 200, ms: 4500, tokens: 2000 };
}
// enough requests that the page scrolls well past the story
const initial = Array.from({ length: 40 }, (_, i) => req(i + 1));
// a failed request from earlier today, outside the live trace: the Usage row's
// View routing opens it through /api/gateway/route
const picked = req(90);
picked.model = "relay/gpt-6-hist";
picked.tries = [{ id: "relay", model: "gpt-6-hist", start: picked.time, done: true, status: 400, ms: 120, error: "Relay: unsupported parameter" }];
picked.status = 400;

function serve(lang, feed) {
  return async (route) => {
    const url = new URL(route.request().url());
    const json = (data) => route.fulfill({ json: data });
    if (url.pathname === "/boot.js") return route.fulfill({ contentType: "text/javascript", body: `window.bootPrefs = {lang:"${lang}",theme:"dark",web:true};` });
    if (url.pathname === "/wails/runtime.js") return route.fulfill({ contentType: "text/javascript", body: "export const Window = {};" });
    if (url.pathname === "/api/state") return json({ agents: [{ id: "codex", name: "Codex", fields: [] }], profiles: [], settings: { lang, theme: "dark" } });
    if (url.pathname === "/api/gateway/trace") {
      const routes = url.searchParams.has("wait") ? await new Promise((resolve) => { feed.next = (value) => { feed.next = null; resolve(value); }; }) : feed.routes || initial;
      return json({ mine: true, now: now.toISOString(), seq: routes.at(-1)?.seq || 1, totals: { requests: 40, rerouted: 0, errors: 0 }, routes });
    }
    if (url.pathname === "/api/gateway/history") return json({ cut: false, days: [{ day, requests: initial.length }], routes: url.searchParams.get("day") ? feed.routes || initial : [] });
    if (url.pathname === "/api/gateway/route" && feed.route) return json(feed.route);
    if (url.pathname === "/api/usage") return json({ calls: 1, errors: 1, input: 0, output: 0, cache_read: 0, cache_write: 0, reasoning: 0, unpriced: 0, cost: 0, bucket: "day", series: [], agents: [], models: [], path: "" });
    if (url.pathname === "/api/usage/quotas") return json([]);
    if (url.pathname === "/api/sessions") return json({ sessions: [], dirs: [] });
    if (url.pathname === "/api/groups") return json({ groups: [] });
    if (url.pathname === "/api/providers") return json({ providers: [], gateway: { running: true, window: true } });
    if (url.pathname.startsWith("/api/")) return json({});
    const file = path.join(assets, url.pathname === "/" ? "index.html" : url.pathname);
    const contentType = { ".html": "text/html", ".js": "text/javascript", ".css": "text/css", ".svg": "image/svg+xml", ".png": "image/png" }[path.extname(file)];
    return route.fulfill({ body: await fs.readFile(file), contentType });
  };
}

const scroll = (page) => page.evaluate(() => document.querySelector("#view-routing").scrollTop);

for (const engine of (process.env.BROWSER ? [process.env.BROWSER] : ["chromium", "webkit"])) {
  for (const lang of ["en", "zh"]) {
    test(`${engine} ${lang}: View routing from Usage brings the picked request's story into view`, async (t) => {
      const browser = await (engine === "webkit" ? webkit.launch() : chromium.launch({ channel: "chromium" }));
      const page = await browser.newPage({ viewport: { width: 1100, height: 800 }, reducedMotion: "reduce" });
      page.setDefaultTimeout(5000);
      const errors = [], feed = { route: picked };
      page.on("pageerror", (e) => errors.push(e.message));
      await page.route("**/*", serve(lang, feed));
      t.after(async () => { feed.next?.([]); await browser.close(); });
      await page.goto("http://magpie.test/?view=routing");
      await page.locator(".rt-req").first().waitFor();

      // the reader had browsed Routing and scrolled down before going to Usage
      await page.mouse.move(550, 400);
      for (let i = 0; i < 40 && (await scroll(page)) < 400; i++) {
        await page.mouse.wheel(0, 300);
        await page.waitForTimeout(40);
      }
      const browsed = await scroll(page);
      assert(browsed >= 400, `the page was scrolled down first (scrollTop ${browsed})`);

      await page.locator('[data-view="usage"]').first().click();
      await page.waitForFunction(() => !document.querySelector("#view-usage").hidden);
      await page.waitForTimeout(300); // the view switch's own settling

      // what the Usage row's View routing button calls
      await page.evaluate((r) => window.openRoute(r.id, r.time), picked);
      await page.waitForFunction(() => document.querySelector(".rt-steps")?.textContent.includes("gpt-6-hist"));
      // let the page's own scrolls settle before judging where the story sits
      let last = -1, same = 0;
      for (let i = 0; i < 60 && same < 4; i++) {
        const s = await scroll(page);
        same = s === last ? same + 1 : 0;
        last = s;
        await page.waitForTimeout(40);
      }
      const seen = await page.evaluate(() => ({ rect: document.querySelector(".rt-log").getBoundingClientRect(), vh: innerHeight }));
      assert(seen.rect.top >= -1 && seen.rect.top <= seen.vh, `the picked request's story is in view (top ${seen.rect.top}px, scroll ${last}px)`);
      assert(seen.rect.bottom > 0, "the story is not wholly above the viewport");
      assert.deepEqual(errors, []);
    });
  }
}
