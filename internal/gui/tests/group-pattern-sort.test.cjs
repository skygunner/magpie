// Run with Node's test runner and Playwright on the module path; see README.md.
// #2730: a model a pattern put in the group had no place of its own — the
// catalog's order decided it, so it had no handle, no Up and no drag, while a
// model added by hand beside it had all three (DeepSeek's own deepseek-flash
// joined a group by a pattern and could not be put above the relay's).
// Moved, it is named among the group's own at the place it was moved to: it
// keeps that place, says it is no longer found by the pattern, can be taken
// out, and is saved with the models the group names. What the patterns still
// add follows them. By the handle's Alt+↑ and by drag, in English and
// Chinese, Chromium and WebKit.
const assert = require("node:assert/strict");
const fs = require("node:fs/promises");
const path = require("node:path");
const { test } = require("node:test");
const { chromium, webkit } = require("playwright");

const assets = path.resolve(__dirname, "../assets");
const models = [
  { id: "an/opus", name: "opus", providerName: "Anthropic", icon: "generic" },
  { id: "or/a:free", name: "a:free", providerName: "OpenRouter", icon: "generic" },
  { id: "or/b:free", name: "b:free", providerName: "OpenRouter", icon: "generic" },
  { id: "or/c:free", name: "c:free", providerName: "OpenRouter", icon: "generic" },
];
const groups = () => ({
  models, pools: [],
  groups: [{ id: "free", name: "Free", members: ["an/opus", "or/a:free", "or/b:free", "or/c:free"], match: ["or/*:free"], matched: ["or/a:free", "or/b:free", "or/c:free"], fast: [], off: [], routing: "order", ready: true,
    patterns: [{ pattern: "or/*:free", models: 3 }],
    memberInfo: ["an/opus", "or/a:free", "or/b:free", "or/c:free"].map((id) => ({ id, ready: true })) }],
});
// every member a pattern's, as a group made around a relay's pattern and then
// given a provider serving the same models has it (the reporter's)
const onlyPatterns = () => ({
  models, pools: [],
  groups: [{ id: "free", name: "Free", members: ["or/a:free", "or/b:free", "or/c:free"], match: ["or/*:free"], matched: ["or/a:free", "or/b:free", "or/c:free"], fast: [], off: [], routing: "order", ready: true,
    patterns: [{ pattern: "or/*:free", models: 3 }],
    memberInfo: ["or/a:free", "or/b:free", "or/c:free"].map((id) => ({ id, ready: true })) }],
});

const words = {
  en: { edit: "Edit", save: "Save", up: "Up", remove: "Remove", by: "by pattern", move: "Move b:free" },
  zh: { edit: "编辑", save: "保存", up: "上移", remove: "移除", by: "按模式匹配", move: "移动 b:free" },
};

function serve(lang, posts, fixture) {
  const state = { agents: [{ id: "codex", name: "Codex", path: "/test/codex", fields: [] }], profiles: [], settings: { lang, theme: "light" } };
  return async (r) => {
    const url = new URL(r.request().url());
    const json = (data) => r.fulfill({ json: data });
    if (url.pathname === "/boot.js") return r.fulfill({ contentType: "text/javascript", body: `window.bootPrefs = {lang:"${lang}",theme:"light",web:true};` });
    if (url.pathname === "/wails/runtime.js") return r.fulfill({ contentType: "text/javascript", body: "export const Window = {};" });
    if (url.pathname === "/api/state") return json(state);
    if (url.pathname === "/api/plugins") return json({ plugins: [] });
    if (url.pathname === "/api/gateway/trace") {
      if (url.searchParams.get("wait")) return new Promise(() => {}); // nothing more comes
      return json({ mine: true, now: new Date().toISOString(), seq: 1, totals: { requests: 0, rerouted: 0, errors: 0 }, routes: [] });
    }
    if (url.pathname === "/api/gateway/history") return json({ cut: false, days: [], routes: [] });
    if (url.pathname === "/api/groups") return json(fixture());
    if (url.pathname.startsWith("/api/groups/")) {
      posts.push({ path: url.pathname, body: JSON.parse(r.request().postData() || "{}") });
      return json(fixture());
    }
    if (url.pathname === "/api/providers") return json({ providers: [], presets: [], excluded: [], gateway: { running: true, window: true, url: "http://127.0.0.1:3999" } });
    if (url.pathname.startsWith("/api/")) return json({});
    const file = path.join(assets, url.pathname === "/" ? "index.html" : url.pathname);
    const contentType = { ".html": "text/html", ".js": "text/javascript", ".css": "text/css", ".svg": "image/svg+xml", ".png": "image/png" }[path.extname(file)];
    await r.fulfill({ body: await fs.readFile(file), contentType });
  };
}

// the editor's members in its own order, and one of them by its name
const order = (ed) => ed.locator(".fallback:not(.rt-pats) .fbrow .n > span:first-child").allTextContents();
const rowOf = (ed, page, name) => ed.locator(".fallback:not(.rt-pats) .fbrow", { has: page.locator(".n > span:first-child", { hasText: new RegExp(`^${name.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")}$`) }) });

for (const engine of (process.env.BROWSER ? [process.env.BROWSER] : ["chromium", "webkit"])) {
  for (const lang of ["en", "zh"]) {
    const w = words[lang];
    test(`${engine} ${lang}: a pattern's model is moved into the group's order`, async (t) => {
      const browser = await (engine === "webkit" ? webkit.launch() : chromium.launch({ channel: "chromium" }));
      const page = await (await browser.newContext({ viewport: { width: 1100, height: 1800 }, reducedMotion: "reduce" })).newPage();
      t.after(() => browser.close());
      page.setDefaultTimeout(5000);
      const errors = [], posts = [];
      page.on("pageerror", (e) => errors.push(e.message));
      await page.route("**/*", serve(lang, posts, groups));
      await page.goto("http://magpie.test/?view=routing");
      const card = page.locator(".rt-group", { hasText: "Free" }).first();
      await card.waitFor();
      await card.locator("button", { hasText: w.edit }).click();
      const ed = page.locator(".rt-gedit");
      await ed.locator(".fbrow").first().waitFor();
      const row = (name) => rowOf(ed, page, name);
      assert.deepEqual(await order(ed), ["opus", "a:free", "b:free", "c:free"]);

      // a model a pattern matched is moved like one added by hand: a handle
      // with its number, and Up
      const handle = row("b:free").locator(".rt-mhandle");
      assert.equal(await handle.count(), 1, "a pattern's model has no handle");
      assert.equal(await handle.getAttribute("aria-label"), w.move);
      assert.equal(await handle.textContent(), "3", "its place in the group");
      assert.equal(await row("b:free").locator("button.text", { hasText: w.up }).count(), 1, "a pattern's model has no Up");

      // Alt+↑ names it above the pattern's first: it stays where it was moved
      await handle.focus();
      await page.keyboard.press("Alt+ArrowUp");
      assert.deepEqual(await order(ed), ["opus", "b:free", "a:free", "c:free"], "the move did not name it");
      assert.equal(await page.evaluate(() => document.activeElement?.closest(".fbrow")?.querySelector(".n > span")?.textContent), "b:free", "the keyboard left the moved model");
      assert(!(await row("b:free").textContent()).includes(w.by), "a moved model still says it is by pattern");
      assert.equal(await row("b:free").locator("button", { hasText: w.remove }).count(), 1, "a moved model can't be taken out");

      // dragged to the top: the handle moves it, as those added by hand
      const hb = await row("a:free").locator(".rt-mhandle").boundingBox(), top = await row("opus").boundingBox();
      await page.mouse.move(hb.x + hb.width / 2, hb.y + hb.height / 2);
      await page.mouse.down();
      for (let i = 1; i <= 8; i++) await page.mouse.move(hb.x + hb.width / 2, hb.y + hb.height / 2 - ((hb.y - top.y + 6) * i) / 8);
      await page.mouse.up();
      assert.deepEqual(await order(ed), ["a:free", "opus", "b:free", "c:free"], "the drag did not move it");

      // saved, the moved models are among those the group names; the one
      // left to the pattern is not, and still follows
      assert((await row("c:free").textContent()).includes(w.by), "the model left to the pattern says so");
      const b = ed.locator("button.primary", { hasText: w.save });
      await b.scrollIntoViewIfNeeded();
      await b.click();
      await page.waitForFunction(() => !document.querySelector(".rt-gedit"));
      const sent = posts.find((p) => p.path === "/api/groups/save");
      assert.deepEqual(sent.body.members, ["or/a:free", "an/opus", "or/b:free"], "the moved models are sent in their order");
      assert.deepEqual(sent.body.match, ["or/*:free"], "the pattern is kept");
      assert.deepEqual(errors, []);
    });

    test(`${engine} ${lang}: a group a pattern filled is put in order by hand`, async (t) => {
      const browser = await (engine === "webkit" ? webkit.launch() : chromium.launch({ channel: "chromium" }));
      const page = await (await browser.newContext({ viewport: { width: 1100, height: 1600 }, reducedMotion: "reduce" })).newPage();
      t.after(() => browser.close());
      page.setDefaultTimeout(5000);
      const errors = [], posts = [];
      page.on("pageerror", (e) => errors.push(e.message));
      await page.route("**/*", serve(lang, posts, onlyPatterns));
      await page.goto("http://magpie.test/?view=routing");
      const card = page.locator(".rt-group", { hasText: "Free" }).first();
      await card.waitFor();
      await card.locator("button", { hasText: w.edit }).click();
      const ed = page.locator(".rt-gedit");
      await ed.locator(".fbrow").first().waitFor();
      assert.deepEqual(await order(ed), ["a:free", "b:free", "c:free"]);
      // with none of its own, Up names the model it is pressed on at the top
      await rowOf(ed, page, "c:free").locator("button.text", { hasText: w.up }).click();
      assert.deepEqual(await order(ed), ["c:free", "a:free", "b:free"]);
      assert((await rowOf(ed, page, "a:free").textContent()).includes(w.by), "the others are still the pattern's");
      const b = ed.locator("button.primary", { hasText: w.save });
      await b.scrollIntoViewIfNeeded();
      await b.click();
      await page.waitForFunction(() => !document.querySelector(".rt-gedit"));
      const sent = posts.find((p) => p.path === "/api/groups/save");
      assert.deepEqual(sent.body.members, ["or/c:free"], "the one moved is the group's own, the rest the pattern's");
      assert.deepEqual(sent.body.match, ["or/*:free"]);
      assert.deepEqual(errors, []);
    });
  }
}
