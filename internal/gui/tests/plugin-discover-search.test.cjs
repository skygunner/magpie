// Run with Node's test runner and Playwright on the module path; see README.md.
// Discover's search finds a plugin by the words its card says (Peterpig:
// 插件、发现页面，搜索opencode，会出现很多别的不相关的供应商). Every community
// plugin's npm package is @magpie-community/opencode-..., the same prefix in
// every one of them, and their one publisher is yetone, so a search for
// opencode (or yetone) answered with the whole market — GitHub Copilot,
// Cursor, Zed, Word Guard — none of which says opencode anywhere. A card
// that does show its package, an unofficial one nobody publishes, is still
// found by it. The entries are magpie-community/plugins' registry.json of
// 2026-10-27 and what npm says of those packages; the API is faked and no
// request leaves the page. English, Chinese, Traditional Chinese, Japanese
// and German, Chromium and WebKit.
const assert = require("node:assert/strict");
const fs = require("node:fs/promises");
const path = require("node:path");
const { test } = require("node:test");
const { chromium, webkit } = require("playwright");

const assets = path.resolve(__dirname, "../assets");

// the market's own entries, as /api/plugins/listings hands them over
const listings = [
  { package: "@magpie-community/opencode-copilot-auth", name: "GitHub Copilot", providers: ["github-copilot"], community: true,
    summary: { en: "GitHub Copilot on GitHub.com or GitHub Enterprise, with premium request or credit usage, remaining allowance and quota reset dates.", zh: "GitHub Copilot：支持 GitHub.com 与 GitHub Enterprise，显示高级请求或积分用量、剩余额度和重置时间。" } },
  { package: "@magpie-community/opencode-zen-free-auth", name: "OpenCode Zen Free", icon: "opencode", providers: ["opencode-zen-free"], community: true,
    summary: { en: "OpenCode Zen's free models with the public credential, automatic discovery and native streaming protocols.", zh: "使用 public 凭证接入 OpenCode Zen 免费模型，自动发现模型并支持原生流式协议。" } },
  { package: "@magpie-community/opencode-zed-auth", name: "Zed", icon: "zed", providers: ["zed"], community: true, replaces: "zed",
    summary: { en: "Claude, GPT, Gemini and Grok hosted by Zed, on your Zed plan.", zh: "用 Zed 订阅使用 Zed 托管的 Claude、GPT、Gemini 和 Grok。" } },
  { package: "@magpie-community/opencode-cursor-auth", name: "Cursor", icon: "cursor", providers: ["cursor"], community: true, replaces: "cursor",
    summary: { en: "Cursor Pro, Pro+, Ultra and Teams — every model your plan has, with tool calls.", zh: "Cursor Pro、Pro+、Ultra 与 Teams：套餐里的所有模型，支持工具调用。" } },
  { package: "@magpie-community/middleware-word-guard", name: "Word Guard", kind: "middleware", community: true,
    summary: { en: "New API's sensitive-word filter: turn away or mask words and patterns in what you send, and in replies.", zh: "New API 的敏感词过滤：拒绝或替换你发出的内容和回复里的词与正则。" } },
  // nobody's community plugin: its card names its package, as the byline
  { package: "opencode-gemini-auth", name: "Gemini", providers: ["google"],
    summary: { en: "Gemini with a Google account.", zh: "用 Google 账号使用 Gemini。" } },
];

// npm's answer for the market's packages: one publisher for all of them
const NPM = Object.fromEntries(listings
  .filter((l) => l.community)
  .map((l) => [l.package, { version: "0.1.0", weekly: 120, publisher: "yetone", license: "MIT" }]));
NPM["opencode-gemini-auth"] = { version: "1.4.0", weekly: 9100, license: "MIT" };

const HITS = [{ package: "opencode-quota", version: "1.0.0", description: "Quota left, in OpenCode's status line", publisher: "someone" }];

function server(lang, fetched) {
  return async (route) => {
    const url = new URL(route.request().url());
    const json = (data) => route.fulfill({ json: data });
    if (url.pathname === "/boot.js") return route.fulfill({ contentType: "text/javascript", body: `window.bootPrefs = {lang:"${lang}",theme:"light",web:false};` });
    if (url.pathname === "/wails/runtime.js") return route.fulfill({ contentType: "text/javascript", body: "export const Window = {};" });
    if (url.pathname === "/api/state") return json({ agents: [], profiles: [], settings: { lang, theme: "light" } });
    if (url.pathname === "/api/providers") return json({ providers: [], presets: [], excluded: [], gateway: { running: true, window: true }, plugins: [] });
    if (url.pathname === "/api/groups") return json({ groups: [], pools: [] });
    if (url.pathname === "/api/plugins") return json({ bun: true, bunVersion: "1.4.2", plugins: [], picker: false, movable: [] });
    if (url.pathname === "/api/plugins/listings") return json({ listings });
    if (url.pathname === "/api/plugins/npm") return json({ npm: NPM });
    if (url.pathname === "/api/plugins/github") return json({ repos: [] });
    if (url.pathname === "/api/plugins/search") return json({ hits: url.searchParams.get("q") === "opencode" ? HITS : [] });
    if (url.pathname.startsWith("/api/")) return json({});
    if (url.host !== "magpie.test" || !/^\/[\w./-]*$/.test(url.pathname)) { fetched.push(url.href); return route.fulfill({ status: 404, body: "" }); }
    const file = path.join(assets, url.pathname === "/" ? "index.html" : url.pathname);
    const contentType = { ".html": "text/html", ".js": "text/javascript", ".css": "text/css", ".svg": "image/svg+xml", ".png": "image/png" }[path.extname(file)];
    try { await route.fulfill({ body: await fs.readFile(file), contentType }); } catch { await route.fulfill({ status: 404, body: "" }); }
  };
}

for (const engine of (process.env.BROWSER ? [process.env.BROWSER] : ["chromium", "webkit"])) {
  test(engine + ": Discover's search answers with the plugins whose cards say what was typed", async (t) => {
    assert(["chromium", "webkit"].includes(engine), "BROWSER must be chromium or webkit");
    const browser = await (engine === "webkit" ? webkit.launch() : chromium.launch({ channel: process.env.PLAYWRIGHT_CHANNEL || "chromium" }));
    t.after(() => browser.close());
    for (const lang of ["en", "zh", "zh-TW", "ja", "de"]) {
      await t.test(lang, async () => {
        const ctx = await browser.newContext({ viewport: { width: 1100, height: 900 } });
        const page = await ctx.newPage();
        page.setDefaultTimeout(6000);
        const errors = [], fetched = [];
        page.on("pageerror", (e) => errors.push(e.message));
        await page.route("**/*", server(lang, fetched));
        try {
          await page.goto("http://magpie.test/?view=plugins");
          const view = page.locator("#view-plugins");
          const cards = view.locator(".pm-body .pm-card");
          await cards.first().waitFor();
          const tr = (s) => page.evaluate((s) => t(s), s);
          const q = view.locator(".pm-head .pm-find input");
          // nothing typed: the community's own, as before
          assert.equal(await cards.count(), 5);
          assert.equal(await view.locator('.pm-card[data-pkg="opencode-gemini-auth"]').count(), 0);

          // the cards the page suggests for the words typed, npm's own
          // section apart, as it comes after them
          let suggested = "";
          const picked = async () => {
            suggested = suggested || (await tr("Suggested"));
            return view.locator(".pm-sec", { has: page.locator(".pm-sechead h3", { hasText: suggested }) }).locator(".pm-card .pm-name b").allInnerTexts();
          };

          // opencode: the cards that say it, OpenCode Zen Free's name and
          // the unofficial one's byline, and npm's own below
          await q.fill("opencode");
          assert.deepEqual(await picked(), ["OpenCode Zen Free", "Gemini"], "opencode leaves the cards that say it");
          // npm's opencode plugins are still offered, where they belong
          await view.locator('.pm-card[data-pkg="opencode-quota"]').waitFor();
          assert.equal(await view.locator(".pm-body .pm-sechead h3", { hasText: await tr("On npm") }).count(), 1);

          // the one publisher the market's packages all have, its cards
          // naming none of it: nothing to suggest
          await q.fill("yetone");
          assert.deepEqual(await picked(), [], "no card says yetone");
          assert.equal(await view.locator(".pm-body .pm-sechead h3", { hasText: suggested }).count(), 0);

          // the words the cards do say: the name
          for (const [typed, name] of [["cursor", "Cursor"], ["zed", "Zed"], ["word guard", "Word Guard"]]) {
            await q.fill(typed);
            assert.deepEqual(await picked(), [name], typed);
          }

          // a card that shows its package, its byline being the package, is
          // found by it
          await q.fill("gemini-auth");
          assert.deepEqual(await picked(), ["Gemini"], "an unofficial card keeps its package as its byline");
          assert.equal(await view.locator('.pm-card[data-pkg="opencode-gemini-auth"] .pm-by').innerText(), "opencode-gemini-auth");

          // Escape puts the community's own back
          await q.press("Escape");
          assert.equal(await q.inputValue(), "");
          await page.waitForFunction(() => document.querySelectorAll("#view-plugins .pm-body .pm-card").length === 5);

          assert.deepEqual(fetched, [], "nothing fetched from elsewhere");
          assert.deepEqual(errors, []);
        } finally {
          await ctx.close();
        }
      });
    }
  });
}
