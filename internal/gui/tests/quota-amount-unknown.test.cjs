// Run with Node's test runner and Playwright on the module path; see README.md.
// #3487: a window the vendor gave a limit but no count of what was used
// (a plugin's "amount": null) is not a full allowance. The card says the
// count is not said — "— / 10,000 Neurons" — beside whatever share it
// knows, never "10,000 / 10,000" with everything left; a window whose
// count is said goes on as it was. Both engines, English and Chinese;
// no backend, the API is faked here.
const assert = require("node:assert/strict");
const fs = require("node:fs/promises");
const path = require("node:path");
const { test } = require("node:test");
const { chromium, webkit } = require("playwright");

const assets = path.resolve(__dirname, "../assets");
const resetsAt = new Date(Date.now() + 12 * 3600e3).toISOString();
const quotas = [
  { provider: "cloudflare-workers-ai", name: "Workers AI", icon: "generic", plan: "Free", user: "a", windows: [
    { name: "Daily", used: 0, resetsAt, limit: 10000, unit: "Neurons", amountUnknown: true },
  ] },
  { provider: "workbuddy", name: "WorkBuddy", icon: "workbuddy-color", plan: "Free", user: "b", windows: [
    { name: "Credits", used: 71, resetsAt, amount: 3550, limit: 5000, unit: "credits" },
  ] },
];

function serve(lang, left) {
  const settings = { lang, theme: "light", quotaLeft: left, currency: "usd" };
  const state = { agents: [], profiles: [], settings };
  return async (r) => {
    const url = new URL(r.request().url());
    const json = (data) => r.fulfill({ json: data });
    if (url.pathname === "/boot.js") return r.fulfill({ contentType: "text/javascript", body: `window.bootPrefs = {lang:"${lang}",theme:"light",web:true};` });
    if (url.pathname === "/wails/runtime.js") return r.fulfill({ contentType: "text/javascript", body: "export const Window = {};" });
    if (url.pathname === "/api/state") return json(state);
    if (url.pathname === "/api/settings") return json(settings);
    if (url.pathname === "/api/gateway/trace") return url.searchParams.get("wait") ? new Promise(() => {}) : json({ mine: true, now: new Date().toISOString(), seq: 0, totals: {}, routes: [] });
    if (url.pathname === "/api/gateway/history") return json({ cut: false, days: [], routes: [] });
    if (url.pathname === "/api/groups") return json({ models: [], groups: [], pools: [] });
    if (url.pathname === "/api/providers") return json({ providers: [], presets: [], excluded: [], gateway: { running: true, window: true, url: "http://127.0.0.1:3999" } });
    if (url.pathname === "/api/usage/quotas") return json(quotas);
    if (url.pathname === "/api/plugins") return json({ plugins: [] });
    if (url.pathname.startsWith("/api/")) return json({});
    const file = path.join(assets, url.pathname === "/" ? "index.html" : url.pathname);
    const contentType = { ".html": "text/html", ".js": "text/javascript", ".css": "text/css", ".svg": "image/svg+xml", ".png": "image/png" }[path.extname(file)];
    try { await r.fulfill({ body: await fs.readFile(file), contentType }); } catch { await r.fulfill({ status: 404, body: "" }); }
  };
}

const squash = (s) => s.replace(/\s+/g, " ").trim();

for (const engine of (process.env.BROWSER ? [process.env.BROWSER] : ["chromium", "webkit"])) {
  for (const lang of ["en", "zh"]) {
    for (const left of [false, true]) {
      test(`${engine} ${lang} ${left ? "left" : "used"}: an unknown count is not a full allowance (#3487)`, async () => {
        const browser = await { chromium, webkit }[engine].launch();
        try {
          const page = await (await browser.newContext({ viewport: { width: 1000, height: 820 }, reducedMotion: "reduce" })).newPage();
          page.setDefaultTimeout(5000);
          await page.route("**/*", serve(lang, left));
          await page.goto("http://magpie.test/?view=usage");
          const cards = page.locator(".subscription-card");
          await cards.nth(1).locator(".quota-n").waitFor();
          const unknown = squash(await cards.first().locator(".quota-n").first().textContent());
          assert.ok(unknown.includes("— / 10,000"), `"${unknown}" says the count is not told`);
          assert.ok(!unknown.includes("10,000 / 10,000"), `"${unknown}" claims the whole allowance`);
          // a count that is said reads as it was
          const known = squash(await cards.nth(1).locator(".quota-n").first().textContent());
          assert.ok(known.startsWith(left ? "1,450 / 5,000" : "3,550 / 5,000"), `"${known}"`);
          await page.close();
        } finally {
          await browser.close();
        }
      });
    }
  }
}
