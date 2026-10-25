// Run with Node's test runner and Playwright on the module path; see README.md.
// The tray panel's way to Plugins opens the main window on Discover, where
// the subscriptions' plugins are, with what was looked for: it asked for
// ?view=plugins alone, so a window that opened with Built-in or Installed as
// its last tab restored that one instead, and a search's own words were lost
// (#3579). The panel has no Plugins tile of its own yet (the Providers page
// draws them), so the call its tile makes is driven here directly. English
// and Chinese; the API is faked here.
const assert = require("node:assert/strict");
const fs = require("node:fs/promises");
const path = require("node:path");
const { test } = require("node:test");
const { chromium, webkit } = require("playwright");

const assets = path.resolve(__dirname, "../assets");

const listings = [
  { package: "@magpie-community/opencode-zed-auth", name: "Zed", icon: "zed", providers: ["zed"], community: true,
    summary: { en: "Your Zed plan.", zh: "你的 Zed 订阅。" }, npm: { version: "0.1.0", weekly: 120 } },
];
const hits = [
  { package: "opencode-fakeco-auth", name: "fakeco", version: "0.2.0", publisher: "fakeco", weekly: 40, description: "a fake" },
];

function server(lang, opened, asked, fetched) {
  return async (route) => {
    const url = new URL(route.request().url());
    const json = (data) => route.fulfill({ json: data });
    if (url.pathname === "/boot.js") return route.fulfill({ contentType: "text/javascript", body: `window.bootPrefs = {lang:"${lang}",theme:"light",web:false};` });
    if (url.pathname === "/wails/runtime.js") return route.fulfill({ contentType: "text/javascript", body: "export const Window = {};" });
    if (url.pathname === "/api/state") return json({ agents: [], profiles: [], settings: { lang, theme: "light" } });
    if (url.pathname === "/api/providers") {
      return json({ providers: [{ id: "openai", name: "OpenAI", icon: "openai", preset: "openai", models: [], agents: [], key: { set: true, masked: "sk-…ab12" } }],
        presets: [], excluded: [], gateway: { running: true, window: true }, plugins: [] });
    }
    if (url.pathname === "/api/groups") return json({ groups: [], pools: [] });
    if (url.pathname === "/api/gateway/trace") return json({ routes: [] });
    if (url.pathname === "/api/plugins") return json({ bun: true, bunVersion: "1.3.0", plugins: [] });
    if (url.pathname === "/api/plugins/listings") return json({ listings });
    if (url.pathname === "/api/plugins/search") { asked.push(url.searchParams.get("q")); return json({ hits }); }
    if (url.pathname === "/api/plugins/builtin") return json([{ id: "usage", name: "Usage", sub: "Your allowances", on: true, chosen: true, source: "preset", state: "active", inUse: 0 }]);
    if (url.pathname === "/api/window/main") { opened.push(url.search); return route.fulfill({ status: 204 }); }
    if (url.pathname.startsWith("/api/")) return json({});
    if (!/^\/[\w./-]*$/.test(url.pathname) || url.host !== "magpie.test") { fetched.push(url.href); return route.fulfill({ status: 404, body: "" }); }
    const file = path.join(assets, url.pathname === "/" ? "index.html" : url.pathname);
    const contentType = { ".html": "text/html", ".js": "text/javascript", ".css": "text/css", ".svg": "image/svg+xml", ".png": "image/png" }[path.extname(file)];
    try { await route.fulfill({ body: await fs.readFile(file), contentType }); } catch { await route.fulfill({ status: 404, body: "" }); }
  };
}

const L = { en: { discover: "Discover", installed: "Installed" }, zh: { discover: "发现", installed: "已安装" } };

for (const engine of (process.env.BROWSER ? [process.env.BROWSER] : ["chromium", "webkit"])) {
  test(engine + ": the panel's Plugins opens Discover, with what was looked for", async (t) => {
    const browser = await (engine === "webkit" ? webkit.launch() : chromium.launch({ channel: "chromium" }));
    t.after(() => browser.close());
    for (const lang of ["en", "zh"]) {
      await t.test(lang, async () => {
        const w = L[lang], opened = [], asked = [], fetched = [], errors = [];
        const context = await browser.newContext({ viewport: { width: 440, height: 700 } });
        // the last tab visited: what a window without a tab opens on
        await context.addInitScript(() => { try { localStorage.setItem("magpie.pluginTab", "installed"); } catch {} });
        const page = await context.newPage();
        page.setDefaultTimeout(5000);
        page.on("pageerror", (e) => errors.push(e.message));
        await page.route("**/*", server(lang, opened, asked, fetched));

        // the tray panel: its way to Plugins, with a search and without
        await page.goto("http://magpie.test/?mode=panel");
        await page.locator("#ptabs").waitFor();
        await page.evaluate(() => openPlugins("fakeco"));
        for (let i = 0; i < 50 && !opened.length; i++) await page.waitForTimeout(20);
        assert.deepEqual(opened, ["?view=plugins&find=fakeco"], "the panel's call carries the search");
        await page.evaluate(() => openPlugins());
        for (let i = 0; i < 50 && opened.length < 2; i++) await page.waitForTimeout(20);
        assert.deepEqual(opened, ["?view=plugins&find=fakeco", "?view=plugins&find="], "and Discover with nothing looked for");

        // the window it asks for, with that last tab still kept: Discover,
        // looking for what the panel asked
        const win = await context.newPage();
        win.setDefaultTimeout(5000);
        win.on("pageerror", (e) => errors.push(e.message));
        await win.route("**/*", server(lang, opened, asked, fetched));
        await win.goto("http://magpie.test/?view=plugins&find=fakeco");
        const view = win.locator("#view-plugins");
        await view.waitFor({ state: "visible" });
        await win.waitForFunction(() => document.querySelector("#view-plugins .lib-tabs .opt[aria-pressed=true]"));
        assert.equal((await view.locator(".lib-tabs .opt[aria-pressed=true]").innerText()).trim(), w.discover, "the tab restored was " + w.installed);
        assert.equal(await view.locator(".pm-find input").inputValue(), "fakeco", "the search the panel carried");
        assert.deepEqual(asked, ["fakeco"], "npm asked for it");
        await view.locator('.pm-card[data-pkg="opencode-fakeco-auth"]').waitFor();
        assert.equal(await view.locator("#builtinList").count(), 0, "not the built-in list");

        assert.deepEqual(fetched, [], "nothing fetched from elsewhere");
        assert.deepEqual(errors, []);
      });
    }
  });
}
