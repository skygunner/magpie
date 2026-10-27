// Run with Node's test runner and Playwright on the module path; see README.md.
// A card names one package. npm's own of a bare name is installed as
// npm:<name>, so resolve keeps it npm's instead of taking it for the
// community plugin the listing is called by (mnsosa's opencode-cursor-auth
// beside @magpie-community/opencode-cursor-auth, #3928), and it installs
// with the community list unread too, as npm: needs no market. The
// community's own scoped package, and a repository installed from GitHub,
// go as they are. The API is faked here.
const assert = require("node:assert/strict");
const fs = require("node:fs/promises");
const path = require("node:path");
const { test } = require("node:test");
const { chromium, webkit } = require("playwright");

const assets = path.resolve(__dirname, "../assets");

const LISTINGS = [
  { package: "@magpie-community/opencode-cursor-auth", name: "Cursor", icon: "cursor", providers: ["cursor"], community: true, replaces: "cursor",
    summary: { en: "Cursor's models on your Cursor plan." }, npm: { version: "0.3.1", weekly: 1200, license: "MIT" } },
];
// tagged magpie-plugin, its author published it to npm from the repository:
// the spec magpie gives is npm's name, bare
const REPOS = [
  { spec: "opencode-kiro-auth", repo: "mnsosa/opencode-kiro-auth", url: "https://github.com/mnsosa/opencode-kiro-auth", owner: "mnsosa",
    description: "Kiro auth for OpenCode.", stars: 12, license: "MIT", pushed: "2026-09-22T23:47:15Z", package: "opencode-kiro-auth", version: "1.0.16" },
];
const HITS = [{ package: "opencode-devin-auth", version: "1.0.16", description: "Devin auth for OpenCode.", publisher: "mnsosa" }];

function server(asked) {
  const installed = [];
  const state = () => ({ bun: true, bunVersion: "1.4.2", plugins: installed, picker: false, movable: [] });
  return async (route) => {
    const url = new URL(route.request().url());
    const json = (data) => route.fulfill({ json: data });
    const body = () => route.request().postDataJSON();
    if (url.pathname === "/boot.js") return route.fulfill({ contentType: "text/javascript", body: `window.bootPrefs = {lang:"en",theme:"light",web:false};` });
    if (url.pathname === "/wails/runtime.js") return route.fulfill({ contentType: "text/javascript", body: "export const Window = {};" });
    if (url.pathname === "/api/state") return json({ agents: [], profiles: [], settings: { lang: "en", theme: "light" } });
    if (url.pathname === "/api/providers") return json({ providers: [], presets: [], excluded: [], gateway: { running: true, window: true }, plugins: [] });
    if (url.pathname === "/api/groups") return json({ groups: [], pools: [] });
    if (url.pathname === "/api/gateway/trace") return json({ routes: [] });
    if (url.pathname === "/api/plugins") return json(state());
    if (url.pathname === "/api/plugins/listings") return json({ listings: LISTINGS });
    if (url.pathname === "/api/plugins/npm") return json({ npm: { "@magpie-community/opencode-cursor-auth": LISTINGS[0].npm } });
    if (url.pathname === "/api/plugins/github") return json({ repos: REPOS, topic: "magpie-plugin" });
    if (url.pathname === "/api/plugins/search") { asked.push(["search", url.searchParams.get("q")]); return json({ hits: HITS }); }
    if (url.pathname === "/api/plugins/add") {
      const b = body();
      asked.push(["add", b]);
      // resolve's answer: npm:<name> installs npm's package <name>
      const spec = b.spec.replace(/^npm:/, "");
      installed.push({ spec, package: spec, providers: [], version: "1.0.16", moved: [] });
      return json(state());
    }
    if (url.pathname.startsWith("/api/")) return json({});
    if (!/^\/[\w./-]*$/.test(url.pathname) || url.host !== "magpie.test") { asked.push(["fetched", url.href]); return route.fulfill({ status: 404, body: "" }); }
    const file = path.join(assets, url.pathname === "/" ? "index.html" : url.pathname);
    const contentType = { ".html": "text/html", ".js": "text/javascript", ".css": "text/css", ".svg": "image/svg+xml", ".png": "image/png" }[path.extname(file)];
    try { await route.fulfill({ body: await fs.readFile(file), contentType }); } catch { await route.fulfill({ status: 404, body: "" }); }
  };
}

for (const engine of (process.env.BROWSER ? [process.env.BROWSER] : ["chromium", "webkit"])) {
  test(engine + ": a card installs npm's own package of a bare name", async (t) => {
    const browser = await (engine === "webkit" ? webkit.launch() : chromium.launch({ channel: process.env.PLAYWRIGHT_CHANNEL || "chromium" }));
    t.after(() => browser.close());
    const asked = [];
    const page = await (await browser.newContext({ viewport: { width: 980, height: 860 } })).newPage();
    page.setDefaultTimeout(5000);
    const errors = [];
    page.on("pageerror", (e) => errors.push(e.message));
    await page.route("**/*", server(asked));
    await page.goto("http://magpie.test/?view=plugins");
    const view = page.locator("#view-plugins");
    await view.waitFor({ state: "visible" });
    // the request the click sends, waited for as it goes out
    const install = async (pkg) => {
      const req = page.waitForRequest((r) => new URL(r.url()).pathname === "/api/plugins/add");
      await view.locator(`.pm-card[data-pkg="${pkg}"] .pm-act`).click();
      return (await req).postDataJSON().spec;
    };

    // the community's own card: its package, scope and all
    assert.equal(await install("@magpie-community/opencode-cursor-auth"), "@magpie-community/opencode-cursor-auth");
    // npm's bare name of a repository's own package: npm's, not a community
    // plugin's called by it
    await view.locator('.pm-card[data-pkg="opencode-kiro-auth"]').waitFor();
    assert.equal(await install("opencode-kiro-auth"), "npm:opencode-kiro-auth");
    // and npm's name a search finds
    await view.locator(".pm-find input").fill("devin");
    await view.locator('.pm-card[data-pkg="opencode-devin-auth"]').waitFor();
    assert.ok(asked.some(([k, v]) => k === "search" && v === "devin"));
    assert.equal(await install("opencode-devin-auth"), "npm:opencode-devin-auth");

    assert.deepEqual(asked.filter(([k]) => k === "fetched"), [], "nothing fetched from outside");
    assert.deepEqual(errors, []);
    await page.context().close();
  });
}
