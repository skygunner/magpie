// Run with Node's test runner and Playwright on the module path; see README.md.
// A plugin's usage window that leaves out used says nothing of how much of
// it is spent — a 0% it never reported. The card shows what the plugin did
// say (its display text) and no percentage, and no meter is drawn from it,
// rather than "100% left" (or NaN%) from 0 it was never told (#3650). A
// window that does say used still shows its share. Chromium and WebKit;
// no backend, the API is faked here.
const assert = require("node:assert/strict");
const fs = require("node:fs/promises");
const path = require("node:path");
const { test } = require("node:test");
const { chromium, webkit } = require("playwright");

const assets = path.resolve(process.env.MAGPIE_ASSETS || path.join(__dirname, "../assets"));

const quotas = [
  // the reporter's plugin: a count in its display, no used
  { provider: "plugin:loomy", name: "Loomy", user: "spent@example.com", windows: [{ name: "Credits", display: "0 / 5000" }] },
  { provider: "claude", name: "Claude", user: "fine@example.com", windows: [{ name: "5 hours", used: 40 }] },
];

function serve(lang) {
  return async (route) => {
    const url = new URL(route.request().url());
    const json = (data) => route.fulfill({ json: data });
    if (url.pathname === "/boot.js") return route.fulfill({ contentType: "text/javascript", body: `window.bootPrefs = {lang:"${lang}",theme:"light",web:true};` });
    if (url.pathname === "/wails/runtime.js") return route.fulfill({ contentType: "text/javascript", body: "export const Window = {};" });
    if (url.pathname === "/api/state") return json({ agents: [], profiles: [], settings: { lang, theme: "light" } });
    if (url.pathname === "/api/providers") return json({ providers: [], presets: [], excluded: [], gateway: { running: true, window: true } });
    if (url.pathname === "/api/usage/quotas") return json(quotas);
    if (url.pathname === "/api/groups") return json({ groups: [], models: [] });
    if (url.pathname === "/api/plugins") return json({ plugins: [] });
    if (url.pathname.startsWith("/api/")) return json({});
    const file = path.join(assets, url.pathname === "/" ? "index.html" : url.pathname);
    const contentType = { ".html": "text/html", ".js": "text/javascript", ".css": "text/css", ".svg": "image/svg+xml", ".png": "image/png" }[path.extname(file)];
    try { await route.fulfill({ body: await fs.readFile(file), contentType }); } catch { await route.fulfill({ status: 404, body: "" }); }
  };
}

for (const engine of (process.env.BROWSER ? [process.env.BROWSER] : ["chromium", "webkit"])) {
  test(`${engine}: a window that didn't say used shows its text, not a percentage`, async (t) => {
    const browser = await (engine === "webkit" ? webkit.launch() : chromium.launch({ channel: "chromium" }));
    t.after(() => browser.close());
    const page = await browser.newPage({ viewport: { width: 900, height: 700 }, reducedMotion: "reduce" });
    page.setDefaultTimeout(5000);
    const errors = [];
    page.on("pageerror", (e) => errors.push(e.message));
    await page.route("**/*", serve("en"));
    await page.goto("http://magpie.test/?view=usage");

    const spent = page.locator(".subscription-card", { hasText: "Loomy" });
    await spent.locator(".quota-n").waitFor();
    const text = await spent.locator(".quota-n").innerText();
    assert.match(text, /0 \/ 5000/, `the plugin's own count: ${text}`);
    assert.doesNotMatch(text, /NaN|%|100% left/, `no share from a used it never sent: ${text}`);
    assert.equal(await spent.locator(".quota-track").count(), 0, "no meter from a share it never sent");

    // a window that does say used keeps its share
    const fine = page.locator(".subscription-card", { hasText: "Claude" });
    await fine.locator(".quota-n").waitFor();
    assert.match(await fine.locator(".quota-n").innerText(), /40%/, "a window that said its share shows it");
    assert.deepEqual(errors, []);
  });
}
