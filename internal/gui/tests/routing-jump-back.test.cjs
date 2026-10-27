// A request opened on Routing from the Usage page's Requests must not cost the
// reader the day, the request and the purpose filter Routing had (#3202).
// Run as described in README.md; isolated API fixtures, no live gateway.
const assert = require("node:assert/strict");
const fs = require("node:fs/promises");
const path = require("node:path");
const { test } = require("node:test");
const { chromium, webkit } = require("playwright");

const assets = process.env.ASSET_DIR || path.resolve(__dirname, "../assets");
const now = new Date();
const day = (d) => [d.getFullYear(), d.getMonth() + 1, d.getDate()].map((v) => String(v).padStart(2, "0")).join("-");
const pastDay = day(new Date(now.getTime() - 7 * 864e5));
const seat = { id: "openai", provider: "openai", name: "OpenAI", kind: "provider", model: "example-model" };
const route = (id, time, kind) => ({ id, seq: id, time, agent: "codex", model: "example-model", provider: "openai", kind,
  session: "test-chat", tokens: 12, cost: 0.01, priced: true, done: true, status: 200, ms: 200, order: [seat],
  tries: [{ id: "openai", model: "example-model", start: time, done: true, status: 200, ms: 200 }] });
// live: two requests today; the day kept: four review turns and one title
const live = [route(101, new Date(now - 1000).toISOString(), "thread_title"), route(100, new Date(now - 2000).toISOString(), "review")];
const past = [route(41, `${pastDay}T10:04:00Z`, "review"), route(40, `${pastDay}T10:03:00Z`, "review"),
  route(39, `${pastDay}T10:02:00Z`, "thread_title"), route(38, `${pastDay}T10:01:00Z`, "review"),
  route(37, `${pastDay}T10:00:00Z`, "review")];
const ledgerRow = (r) => ({ route_id: r.id, t: r.time, agent: "codex", agentName: "Codex", icon: "codex-color",
  provider: "openai", providerName: "OpenAI", model: r.model, req: r.model, served: r.model, kind: r.kind,
  in: 10, out: 2, cache_read: 3, cache_write: 1, ms: 200, status: 200, cost: 0.01, priced: true });
const ledger = { rows: [ledgerRow(live[0])], offset: 0, total: 1, calls: 1, errors: 0, input: 10, output: 2,
  cache_read: 3, cache_write: 1, cost: 0.01, unpriced: 0, agents: [{ id: "codex", name: "Codex", icon: "codex-color" }],
  purposes: ["kind:thread_title", "kind:review"] };

function server(lang) {
  return async (r) => {
    const req = r.request(), url = new URL(req.url());
    const json = (data) => r.fulfill({ json: data });
    if (url.pathname === "/boot.js") return r.fulfill({ contentType: "text/javascript", body: `window.bootPrefs={lang:"${lang}",theme:"light",web:false};` });
    if (url.pathname === "/wails/runtime.js") return r.fulfill({ contentType: "text/javascript", body: "export const Window={};" });
    if (url.pathname === "/api/state") return json({ agents: [{ id: "codex", name: "Codex", path: "/fixture/config.toml", fields: [] }], profiles: [], settings: { lang, theme: "light" } });
    if (url.pathname === "/api/providers") return json({ providers: [], presets: [], excluded: [], models: [], gateway: { running: true, window: true } });
    if (url.pathname === "/api/groups") return json({ groups: [], models: [] });
    if (url.pathname === "/api/gateway/trace") {
      if (url.searchParams.has("wait")) await new Promise((res) => setTimeout(res, 2000));
      return json({ mine: true, seq: 1, now: now.toISOString(), totals: { requests: 2, rerouted: 0, errors: 0 }, routes: live });
    }
    if (url.pathname === "/api/gateway/history") {
      const d = url.searchParams.get("day");
      return json({ cut: false, days: [{ day: pastDay, requests: past.length }], routes: d ? past : [] });
    }
    if (url.pathname === "/api/usage/requests") return json(ledger);
    if (url.pathname === "/api/usage/quotas") return json([]);
    if (url.pathname.startsWith("/api/")) return json({});
    const file = path.join(assets, url.pathname === "/" ? "index.html" : url.pathname);
    const contentType = { ".html": "text/html", ".js": "text/javascript", ".css": "text/css", ".svg": "image/svg+xml", ".png": "image/png" }[path.extname(file)];
    await r.fulfill({ body: await fs.readFile(file), contentType });
  };
}

for (const engine of (process.env.BROWSER ? [process.env.BROWSER] : ["chromium", "webkit"])) {
  for (const lang of ["en", "zh"]) {
    test(`${engine} ${lang}: a request opened from Usage gives back the day, request and purpose Routing left`, async (t) => {
      const browser = await (engine === "webkit" ? webkit.launch() : chromium.launch({ channel: "chromium" }));
      t.after(() => browser.close());
      const page = await browser.newPage({ viewport: { width: 1100, height: 800 }, reducedMotion: "reduce" });
      page.setDefaultTimeout(7000);
      const errors = [];
      page.on("pageerror", (e) => { if (!/^ResizeObserver loop completed with undelivered notifications/.test(e.message)) errors.push(e.message); });
      await page.route("**/*", server(lang));
      const rows = (n) => page.waitForFunction((n) => document.querySelectorAll("#view-routing .rt-req").length === n, n);
      const label = (s) => page.evaluate((s) => t(s), s);
      await page.goto("http://magpie.test/?view=routing");
      await rows(2);
      // a kept day, its requests narrowed by purpose, and one of them picked
      await page.locator(".rt-days .rt-day").nth(1).click();
      await rows(past.length);
      const review = await page.evaluate(() => purposeOptions(["kind:review"])[0].name);
      await page.locator("#rtPurpose").click();
      await page.locator(".rt-purpose-menu").getByRole("menuitemcheckbox", { name: review, exact: true }).click();
      await page.keyboard.press("Escape");
      const purposeLabel = await page.evaluate((n) => t("Purpose: {name}", { name: n }), review);
      await page.waitForFunction((l) => document.querySelector("#rtPurpose span").textContent === l, purposeLabel);
      await rows(4);
      await page.locator(".rt-req").nth(0).click();
      await page.locator(".rt-log-head > span").first().waitFor();
      const picked = await page.locator(".rt-log-head > span").first().textContent();
      assert.notEqual(picked, await label("How the last request was routed"), "the request picked is its own in the log");
      // the Usage page's Requests open that request's routing
      await page.locator('#nav button[data-view="usage"]').click();
      await page.locator('.nav-sub[data-view="usage"] button').filter({ hasText: await label("Requests") }).click();
      await page.locator(".led-row .led-route-link").first().click();
      await page.locator("#view-routing").waitFor({ state: "visible" });
      await rows(2);
      assert.equal(await page.locator("#rtPurpose span").textContent(), await label("Purpose filter"), "the jump dropped the purpose filter");
      assert.equal(await page.locator(".rt-days .rt-day").nth(0).getAttribute("aria-pressed"), "true", "the jump left the kept day");
      // and the page says how to get the three back
      const back = page.getByRole("button", { name: await label("Back to the routing you left"), exact: true });
      await back.click();
      await page.waitForFunction((l) => document.querySelector("#rtPurpose span").textContent === l, purposeLabel);
      await rows(4);
      assert.equal(await page.locator("#rtPurpose span").textContent(), purposeLabel, "the purpose filter is back");
      assert.equal(await page.locator(".rt-days .rt-day").nth(1).getAttribute("aria-pressed"), "true", "the kept day is back");
      assert.equal(await page.locator(".rt-days .rt-day").nth(0).getAttribute("aria-pressed"), "false");
      assert.equal(await page.locator(".rt-log-head > span").first().textContent(), picked, "the request picked is back in the log");
      assert.equal(await back.count(), 0, "one return is all it offers");
      assert.deepEqual(errors, []);
    });
  }
}
