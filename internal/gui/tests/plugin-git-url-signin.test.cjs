// Run with Node's test runner and Playwright on the module path; see README.md.
// A repository tagged magpie-plugin installed by its GitHub page's URL (JoeyWu-tech,
// #3930): the market's card knows it as github:owner/repo, while the installed
// entry and the provider's sign-ins are named by the spec it was installed
// with (https://github.com/owner/repo). The card must offer Sign in the way
// the Installed tab's row does, by the entry's own spec — not sit at
// Installed with the provider unsigned in. English; the API is faked here.
const assert = require("node:assert/strict");
const fs = require("node:fs/promises");
const path = require("node:path");
const { test } = require("node:test");
const { chromium, webkit } = require("playwright");

const assets = path.resolve(__dirname, "../assets");

const REPO = { spec: "github:owner/repo", repo: "owner/repo", url: "https://github.com/owner/repo", owner: "owner", description: "A plugin with a provider to sign in to.", stars: 5, license: "MIT", pushed: "2026-10-01T00:00:00Z", package: "", version: "" };
const HAVE = "c5949841c1e6c8f3ba9f6dc0e2a1b2c3d4e5f607";
// installed by the repository's page's URL, not by github:owner/repo
const INSTALLED = { spec: "https://github.com/owner/repo", package: "my-oc-plugin", providers: ["My Provider"], version: "1.0.0", commit: HAVE, moved: [] };

function server(lang) {
  const state = () => ({ bun: true, bunVersion: "1.4.2", plugins: [INSTALLED], picker: false, movable: [] });
  return async (route) => {
    const url = new URL(route.request().url());
    const json = (data) => route.fulfill({ json: data });
    if (url.pathname === "/boot.js") return route.fulfill({ contentType: "text/javascript", body: `window.bootPrefs = {lang:"${lang}",theme:"light",web:false};` });
    if (url.pathname === "/wails/runtime.js") return route.fulfill({ contentType: "text/javascript", body: "export const Window = {};" });
    if (url.pathname === "/api/state") return json({ agents: [], profiles: [], settings: { lang, theme: "light" } });
    if (url.pathname === "/api/providers") return json({
      providers: [], presets: [], excluded: [], gateway: { running: true, window: true },
      // the plugin's provider checks in under the spec it was installed with
      plugins: [{ id: "my-oc-plugin-my-provider", pid: "my-provider", name: "My Provider", spec: INSTALLED.spec, signedIn: false, models: 3 }],
    });
    if (url.pathname === "/api/groups") return json({ groups: [], pools: [] });
    if (url.pathname === "/api/gateway/trace") return json({ routes: [] });
    if (url.pathname === "/api/plugins") return json(state());
    if (url.pathname === "/api/plugins/listings") return json({ listings: [] });
    if (url.pathname === "/api/plugins/github") return json({ repos: [REPO], topic: "magpie-plugin" });
    if (url.pathname === "/api/plugins/search") return json({ hits: [] });
    if (url.pathname === "/api/plugins/updates") return json({ waiting: [], updated: [] });
    if (url.pathname === "/api/plugins/git") return json({ heads: { [INSTALLED.spec]: HAVE } });
    if (url.pathname === "/api/plugins/page") return json({ readme: "# my-oc-plugin\n\nA plugin.", updated: "2026-10-01T00:00:00Z" });
    if (url.pathname === "/api/open") return route.fulfill({ status: 204 });
    if (url.pathname.startsWith("/api/")) return json({});
    if (!/^\/[\w./-]*$/.test(url.pathname) || url.host !== "magpie.test") return route.fulfill({ status: 404, body: "" });
    const file = path.join(assets, url.pathname === "/" ? "index.html" : url.pathname);
    const contentType = { ".html": "text/html", ".js": "text/javascript", ".css": "text/css", ".svg": "image/svg+xml", ".png": "image/png" }[path.extname(file)];
    try { await route.fulfill({ body: await fs.readFile(file), contentType }); } catch { await route.fulfill({ status: 404, body: "" }); }
  };
}

for (const engine of (process.env.BROWSER ? [process.env.BROWSER] : ["chromium", "webkit"])) {
  test(engine + ": a repository's card offers Sign in when installed by its URL", async (t) => {
    const browser = await (engine === "webkit" ? webkit.launch() : chromium.launch({ channel: process.env.PLAYWRIGHT_CHANNEL || "chromium" }));
    t.after(() => browser.close());
    const page = await (await browser.newContext({ viewport: { width: 980, height: 820 } })).newPage();
    page.setDefaultTimeout(5000);
    const errors = [];
    page.on("pageerror", (e) => errors.push(e.message));
    await page.route("**/*", server("en"));

    // Discover: the repository's card says Installed by the URL's spec, and offers Sign in
    await page.goto("http://magpie.test/?view=plugins");
    const view = page.locator("#view-plugins");
    await view.waitFor({ state: "visible" });
    const sec = view.locator(".pm-sec", { has: page.locator("h3", { hasText: "Unofficial, on GitHub" }) });
    await sec.waitFor();
    const card = sec.locator(".pm-card[data-pkg='github:owner/repo']");
    await card.waitFor();
    assert.equal(await card.locator(".pm-act").innerText(), "Sign in", "#3930: the card offers Sign in, as the Installed row does");

    // the Installed tab's row shows the provider unsigned in: the fixture's other half
    await page.addInitScript(() => { try { localStorage.setItem("magpie.pluginTab", "installed"); } catch {} });
    await page.reload();
    const row = view.locator(".pm-row", { hasText: "my-oc-plugin" });
    await row.waitFor();
    assert.match(await row.innerText(), /My Provider: not signed in/);

    assert.deepEqual(errors, []);
  });
}
