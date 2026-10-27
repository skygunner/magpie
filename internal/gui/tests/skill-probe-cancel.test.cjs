// Run with Node's test runner and Playwright on the module path; see README.md.
// #3163: cancelling the skill probe keeps the source in the add box, and the
// Find skills button never sits enabled over an empty box: cancelling then
// clicking Find skills lists the source's skills again, without retyping the
// address. In English and Chinese, in Chromium and WebKit. No backend: the
// API is faked here.
const assert = require("node:assert/strict");
const fs = require("node:fs/promises");
const path = require("node:path");
const { test } = require("node:test");
const { chromium, webkit } = require("playwright");

const assets = path.resolve(__dirname, "../assets");
const HOME = "/Users/cancel";
const agent = (id, name) => ({ id, name, icon: "", skills: `${HOME}/.${id}/skills` });
const SOURCE = "http://gitlab.example.internal:8081/group/project.git";
const lib = () => ({
  dir: `${HOME}/.magpie/library`, backups: `${HOME}/.magpie/backups`, home: HOME,
  agents: [agent("claude", "Claude Code")],
  instructions: { agents: [] }, servers: [], projects: [], foundSkills: [], newSkills: [],
  checkDue: false,
  skills: [],
});
const probe = { source: SOURCE, kind: "git", candidates: [{ path: "skills/a", name: "a", description: "a", have: false }] };

function server(lang, posts) {
  return async (route) => {
    const req = route.request(), url = new URL(req.url());
    if (url.pathname === "/boot.js") return route.fulfill({ contentType: "text/javascript", body: `window.bootPrefs = {lang:"${lang}",theme:"light",web:true};` });
    if (url.pathname === "/wails/runtime.js") return route.fulfill({ contentType: "text/javascript", body: "export const Window = {};" });
    if (url.pathname === "/api/state") return route.fulfill({ json: { agents: [], profiles: [], settings: { lang, theme: "light" } } });
    if (url.pathname === "/api/library") return route.fulfill({ json: lib() });
    if (url.pathname === "/api/library/skills/probe") { posts.push(req.postDataJSON()); return route.fulfill({ json: probe }); }
    if (url.pathname.startsWith("/api/library/skills/")) { posts.push(url.pathname.split("/").pop()); return route.fulfill({ json: { ...lib(), result: { changed: [] } } }); }
    if (url.pathname === "/api/plugins") return route.fulfill({ json: { plugins: [] } });
    if (url.pathname === "/api/usage/quotas") return route.fulfill({ json: [] });
    if (url.pathname === "/api/groups") return route.fulfill({ json: { groups: [] } });
    if (url.pathname === "/api/providers") return route.fulfill({ json: { providers: [], gateway: { running: true } } });
    if (url.pathname.startsWith("/api/")) return route.fulfill({ json: {} });
    const file = path.join(assets, url.pathname === "/" ? "index.html" : url.pathname);
    const contentType = { ".html": "text/html", ".js": "text/javascript", ".css": "text/css", ".svg": "image/svg+xml", ".png": "image/png" }[path.extname(file)];
    await route.fulfill({ body: await fs.readFile(file), contentType });
  };
}

const words = {
  en: { find: "Find skills", cancel: "Cancel" },
  zh: { find: "查找技能", cancel: "取消" },
};

for (const engine of (process.env.BROWSER ? [process.env.BROWSER] : ["chromium", "webkit"])) {
  test(engine + ": cancelling the skill probe keeps the source and Find skills never fails silently", async (t) => {
    assert(["chromium", "webkit"].includes(engine), "BROWSER must be chromium or webkit");
    const browser = await (engine === "webkit" ? webkit.launch() : chromium.launch({ channel: "chromium" }));
    const errors = [];
    t.after(async () => {
      if (errors.length) console.log(errors);
      await browser.close();
    });
    for (const lang of ["en", "zh"]) {
      await t.test(`${lang}`, async () => {
        const w = words[lang], posts = [];
        const ctx = await browser.newContext({ viewport: { width: 900, height: 900 } });
        await ctx.addInitScript(() => { try { localStorage.setItem("magpie.libTab", "skills"); } catch {} });
        const page = await ctx.newPage();
        page.setDefaultTimeout(5000);
        page.on("pageerror", (e) => errors.push(e.message));
        await page.route("**/*", server(lang, posts));
        await page.goto("http://magpie.test/");
        await page.locator('button[data-view="library"]').click();
        const v = page.locator("#view-library");
        const src = v.locator('input[data-lib="source"]');
        await src.waitFor();

        // probe a source and cancel: the address stays in the box
        await src.fill(SOURCE);
        const find = v.locator(".lib-find button", { hasText: w.find });
        await find.click();
        await v.locator(".lib-probe").waitFor();
        assert.deepEqual(posts.map((p) => p.source), [SOURCE]);
        await v.locator(".lib-probefoot button", { hasText: w.cancel }).click();
        await v.locator(".lib-probe").waitFor({ state: "detached" });
        assert.equal(await src.inputValue(), SOURCE, "cancelling cleared the source box");
        assert.equal(posts.length, 1, "cancelling probed again by itself");

        // Find skills reopens the panel from the kept address
        await find.click();
        await v.locator(".lib-probe").waitFor();
        assert.deepEqual(posts.map((p) => p.source), [SOURCE, SOURCE]);
        await v.locator(".lib-probefoot button", { hasText: w.cancel }).click();
        await v.locator(".lib-probe").waitFor({ state: "detached" });

        // an empty box never offers a silent Find skills
        await src.fill("");
        assert.equal(await find.isDisabled(), true, "Find skills is enabled over an empty box");
        await ctx.close();
      });
    }
    assert.deepEqual(errors, []);
  });
}
