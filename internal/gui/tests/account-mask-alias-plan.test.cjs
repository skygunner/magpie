// #3656: aliases and plan labels stay readable while raw identities are hidden.
const assert = require("node:assert/strict");
const fs = require("node:fs/promises");
const path = require("node:path");
const { test } = require("node:test");
const { chromium, webkit } = require("playwright");

const assets = path.resolve(__dirname, "../assets");
const quotas = [
  { provider: "antigravity", name: "Antigravity", plan: "Google AI Pro", user: "Google", alias: "Work laptop", windows: [{ name: "Credits", used: 20 }] },
  { provider: "antigravity", name: "Antigravity", plan: "Google AI Pro", user: "raw@example.com", alias: "Google", windows: [{ name: "Credits", used: 30 }] },
];
function serve(lang) {
  return async (route) => {
    const url = new URL(route.request().url());
    const json = (data) => route.fulfill({ json: data });
    if (url.pathname === "/boot.js") return route.fulfill({ contentType: "text/javascript", body: `window.bootPrefs = {lang:${JSON.stringify(lang)},theme:"light",web:true};` });
    if (url.pathname === "/wails/runtime.js") return route.fulfill({ contentType: "text/javascript", body: "export const Window = {};" });
    if (url.pathname === "/api/state") return json({ agents: [], profiles: [], settings: { lang, theme: "light" } });
    if (url.pathname === "/api/providers") return json({ providers: [], presets: [], excluded: [], gateway: { running: true, window: true } });
    if (url.pathname === "/api/usage/quotas") return json(quotas);
    if (url.pathname === "/api/groups") return json({ groups: [], models: [] });
    if (url.pathname.startsWith("/api/")) return json({});
    const file = path.join(assets, url.pathname === "/" ? "index.html" : url.pathname);
    const contentType = { ".html": "text/html", ".js": "text/javascript", ".css": "text/css", ".svg": "image/svg+xml", ".png": "image/png" }[path.extname(file)];
    await route.fulfill({ body: await fs.readFile(file), contentType });
  };
}
for (const engine of (process.env.BROWSER ? [process.env.BROWSER] : ["chromium", "webkit"])) {
  for (const lang of ["en", "zh", "zh-TW", "ja", "de"]) {
    for (const width of [1100, 420]) {
      test(`${engine} ${lang} ${width}: masking keeps aliases and plans readable`, async (t) => {
        // Given account aliases and an identity that also names a plan's brand.
        const browser = await (engine === "webkit" ? webkit.launch() : chromium.launch({ channel: "chromium" }));
        t.after(() => browser.close());
        const page = await browser.newPage({ viewport: { width, height: 800 }, reducedMotion: "reduce" });
        page.setDefaultTimeout(5000);
        await page.route("**/*", serve(lang));
        await page.goto("http://magpie.test/?view=usage");
        await page.locator(".subscription-account .quota-alias").first().waitFor();
        const original = await page.locator(".subscription-account").allTextContents();
        await page.evaluate(() => {
          noteAccounts({ alias: "Alias only" });
          const sample = el("div");
          sample.id = "providerSample";
          sample.append(accountNameButton({ accountNames: { google: "Google" } }, "Google"), planSpan({ plan: "Google AI Pro" }), el("span", "model-value", "Google Gemini"), el("span", "", "Alias only"));
          document.querySelector("#view-providers").append(sample);
        });
        // When the real Hide accounts control is enabled.
        await page.locator("#usageMask").click();
        // Then only the original account identities are hidden.
        const got = await page.locator(".subscription-account").evaluateAll((rows) => rows.map((r) => ({
          alias: r.querySelector(".quota-alias").textContent,
          aliasMasked: !!r.querySelector(".quota-alias .pii"),
          user: r.querySelector(".quota-of .pii")?.dataset.raw,
          plan: r.querySelector(".plan").textContent,
          title: r.querySelector(".quota-of").title,
        })));
        assert.deepEqual(got.map((r) => r.alias), ["Work laptop", "Google"]);
        assert.deepEqual(got.map((r) => r.aliasMasked), [false, false]);
        assert.deepEqual(got.map((r) => r.user), ["Google", "raw@example.com"]);
        assert.deepEqual(got.map((r) => r.plan), ["Google AI Pro", "Google AI Pro"]);
        assert.ok(got.every((r) => !/Google|raw@example/.test(r.title)));
        assert.equal(await page.locator("#providerSample .rename").textContent(), "Google");
        assert.equal(await page.locator("#providerSample .acct-user .pii").getAttribute("data-raw"), "Google");
        assert.equal(await page.locator("#providerSample .plan").textContent(), "Google AI Pro");
        assert.equal(await page.locator("#providerSample .model-value").textContent(), "Google Gemini");
        assert.ok((await page.locator("#providerSample").textContent()).includes("Alias only"));
        await page.locator("#usageMask").click();
        assert.deepEqual(await page.locator(".subscription-account").allTextContents(), original);
      });
    }
  }
}
