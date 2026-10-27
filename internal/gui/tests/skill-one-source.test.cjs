// Run with Node's test runner and Playwright on the module path; see README.md.
// A library whose skills all came from one repository (#3122, chengbot-ai):
// By source names that repository above them, with its agent chips and its
// fold, as a group of a library with several sources has. It was a bare
// list, so the source was nowhere on the page and the group couldn't be
// folded. A→Z is still one flat list, with no source to name. English and
// Chinese; the API is faked here.
const assert = require("node:assert/strict");
const fs = require("node:fs/promises");
const path = require("node:path");
const { test } = require("node:test");
const { chromium, webkit } = require("playwright");

const assets = path.resolve(__dirname, "../assets");
const HOME = "/Users/me";
const REPO = "gitlab.example.internal/group/project";
const agent = (id, name, icon) => ({ id, name, icon, skills: `${HOME}/.${id}/skills`, mcp: `${HOME}/.${id}/mcp.json` });
// installed from a self-hosted git host, as the reporter's 12 were
const git = (name) => ({ name, kind: "git", source: `https://${REPO}/-/tree/main/skills/${name}`, repo: REPO, description: "d", agents: ["claude"] });

function server(lang, state) {
  return async (route) => {
    const req = route.request();
    const url = new URL(req.url());
    const json = (data) => route.fulfill({ json: data });
    if (url.pathname === "/boot.js") return route.fulfill({ contentType: "text/javascript", body: `window.bootPrefs = {lang:"${lang}",theme:"light",web:true};` });
    if (url.pathname === "/wails/runtime.js") return route.fulfill({ contentType: "text/javascript", body: "export const Window = {};" });
    if (url.pathname === "/api/state") return json({ agents: [], profiles: [], settings: { lang, theme: "light" } });
    if (url.pathname === "/api/library") return json(state.lib);
    if (url.pathname === "/api/providers") return json({ providers: [], presets: [], excluded: [], gateway: { running: true, window: true }, plugins: [] });
    if (url.pathname === "/api/groups") return json({ groups: [], pools: [] });
    if (url.pathname === "/api/usage/quotas") return json([]);
    if (url.pathname.startsWith("/api/")) return json({});
    if (url.host !== "magpie.test") return route.fulfill({ status: 404, body: "" });
    const file = path.join(assets, url.pathname === "/" ? "index.html" : url.pathname);
    const contentType = { ".html": "text/html", ".js": "text/javascript", ".css": "text/css", ".svg": "image/svg+xml", ".png": "image/png" }[path.extname(file)];
    try { await route.fulfill({ body: await fs.readFile(file), contentType }); } catch { await route.fulfill({ status: 404, body: "" }); }
  };
}

const L = { en: { three: "3 skills" }, zh: { three: "3 个技能" } };

for (const engine of (process.env.BROWSER ? [process.env.BROWSER] : ["chromium", "webkit"])) {
  test(engine + ": one source's skills are named above them, with its chips and its fold", async (t) => {
    assert(["chromium", "webkit"].includes(engine), "BROWSER must be chromium or webkit");
    const browser = await (engine === "webkit" ? webkit.launch() : chromium.launch({ channel: process.env.PLAYWRIGHT_CHANNEL || "chromium" }));
    t.after(() => browser.close());
    for (const lang of Object.keys(L)) {
      await t.test(lang, async () => {
        const state = {
          lib: {
            dir: `${HOME}/.magpie/library`, backups: `${HOME}/.magpie/backups`, home: HOME,
            agents: [agent("claude", "Claude Code", "claudecode-color"), agent("codex", "Codex", "openai")],
            instructions: { agents: [], sets: [] }, servers: [], foundServers: [], projects: [], foundSkills: [], problems: [], skillGroups: [],
            skills: ["grill", "prd", "tdd"].map(git),
          },
        };
        const ctx = await browser.newContext({ viewport: { width: 1100, height: 760 }, reducedMotion: "reduce" });
        await ctx.addInitScript(() => { try { localStorage.setItem("magpie.libTab", "skills"); } catch {} });
        const page = await ctx.newPage();
        page.setDefaultTimeout(5000);
        const errors = [];
        page.on("pageerror", (e) => errors.push(e.message));
        await page.route("**/*", server(lang, state));
        await page.goto("http://magpie.test/");
        await page.locator('button[data-view="library"]').click();
        const v = page.locator("#view-library");
        const rows = () => v.locator(".lib-skill .name").allTextContents();
        await v.locator(".lib-skill").first().waitFor();

        // the one source is named, its three under it, and its chips say
        // which agents have them
        const group = v.locator(".lib-group");
        assert.equal(await group.count(), 1, "one source's skills are one group card");
        const head = group.locator(".lib-grouphead");
        assert.equal((await head.locator(".who .name").textContent()).trim(), REPO);
        assert.equal((await head.locator(".sub").textContent()).trim(), L[lang].three);
        assert.deepEqual((await rows()).map((n) => n.trim()).sort(), ["grill", "prd", "tdd"]);
        const chips = head.locator(".lib-groupchips");
        assert.equal(await chips.locator('.lib-ag[data-agent="claude"]').getAttribute("aria-pressed"), "true");
        assert.equal(await chips.locator('.lib-ag[data-agent="codex"]').getAttribute("aria-pressed"), "false");

        // the heading folds it, and keeps that
        await head.locator(".who .name").click();
        await page.waitForFunction(() => !document.querySelector("#view-library .lib-grouphead").classList.contains("open"));
        assert.deepEqual(await rows(), []);
        const kept = await page.evaluate(() => JSON.parse(localStorage.getItem("magpie.libSkillFolds") || "{}"));
        assert.deepEqual(kept, { ["gh:" + REPO]: true });
        await head.locator(".who .name").click();
        await v.locator(".lib-skill").first().waitFor();
        assert.deepEqual((await rows()).map((n) => n.trim()).sort(), ["grill", "prd", "tdd"]);

        // A→Z is still one flat list, with no source named above it
        const sort = page.locator('.segs.sortby[data-sort="libSkills"]');
        await sort.locator(".opt").nth(1).click();
        await page.waitForFunction(() => !document.querySelector("#view-library .lib-grouphead"));
        assert.deepEqual((await rows()).map((n) => n.trim()).sort(), ["grill", "prd", "tdd"]);

        // and By source brings the heading back
        await sort.locator(".opt").nth(0).click();
        await page.waitForFunction(() => document.querySelector("#view-library .lib-grouphead .name")?.textContent === "gitlab.example.internal/group/project");
        assert.deepEqual(errors, []);
        await ctx.close();
      });
    }
  });
}
