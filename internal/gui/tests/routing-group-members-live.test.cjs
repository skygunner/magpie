// Run with Node's test runner and Playwright on the module path; see README.md.
// The story names the routing group a request asked for and, after the group,
// its models. A group's list edited after the request went was not the one it
// named: the sentence kept the models the route was planned with, so a removed
// model was still listed and an added one missing, through leaving the page
// and coming back, until magpie restarted (#3156, AvatarGanymede). It now names
// the models the group has, as the groups below the requests name them, and a
// save is told at once, without another request routed. English and Chinese,
// Chromium and WebKit.
const assert = require("node:assert/strict");
const fs = require("node:fs/promises");
const path = require("node:path");
const { test } = require("node:test");
const { chromium, webkit } = require("playwright");

const assets = path.resolve(process.env.MAGPIE_ROUTING_ASSETS || path.join(__dirname, "../assets"));
const models = [
  { id: "a/m", name: "MA", providerName: "A", icon: "generic" },
  { id: "b/m", name: "MB", providerName: "B", icon: "generic" },
  { id: "c/luna", name: "Luna", providerName: "C", icon: "generic" },
];
const info = (ids) => ids.map((id) => {
  const m = models.find((x) => x.id === id);
  return { id, ready: true, name: m.name, provider: m.id.split("/")[0], model: m.name, icon: "generic" };
});
const group = (members) => ({
  id: "hand", name: "Hand", routing: "", affinity: "turn", ready: true,
  members, memberInfo: info(members), holds: [], offers: [], shared: [], rules: [],
});
const seat = (p, ms, name) => ({ id: p, provider: p, name, kind: "provider", model: ms, routing: "" });
const when = new Date(Date.now() - 60000).toISOString();
// the request as the trace has it: planned with a/m and b/m when it went
const route = {
  id: 77, seq: 1, time: when, agent: "claude", model: "group/hand",
  order: [seat("a", "m", "A"), seat("b", "m", "B")],
  tries: [{ id: "a", model: "m", start: when, done: true, status: 200, ms: 800 }],
  done: true, status: 200, ms: 800, tokens: 120, out: 20,
  group: { id: "hand", ref: "group/hand", name: "Hand", routing: "", affinity: "turn", members: ["a/m", "b/m"], via: ["", ""] },
};

// serve answers the page: the group's list is the one saved last, so a save
// changes what /api/groups says, as the real one does
function serve(lang, saved) {
  const state = { agents: [{ id: "claude", name: "Claude Code", path: "/test/claude", fields: [] }], profiles: [], settings: { lang, theme: "light" } };
  let groups = { models, pools: [], deciders: [], found: false, groups: [group(["a/m", "c/luna"])] };
  return async (r) => {
    const url = new URL(r.request().url());
    const json = (d) => r.fulfill({ json: d });
    if (url.pathname === "/boot.js") return r.fulfill({ contentType: "text/javascript", body: `window.bootPrefs = {lang:"${lang}",theme:"light",web:true};` });
    if (url.pathname === "/wails/runtime.js") return r.fulfill({ contentType: "text/javascript", body: "export const Window = {};" });
    if (url.pathname === "/api/state") return json(state);
    if (url.pathname === "/api/plugins") return json({ plugins: [] });
    if (url.pathname === "/api/gateway/trace") {
      if (url.searchParams.get("wait")) return new Promise(() => {}); // nothing more comes
      return json({ mine: true, now: new Date().toISOString(), seq: 1, totals: { requests: 1, rerouted: 0, errors: 0 }, routes: [route] });
    }
    if (url.pathname === "/api/gateway/history") return json({ cut: false, days: [], routes: [] });
    if (url.pathname === "/api/groups") return json(groups);
    if (url.pathname === "/api/groups/save") {
      const body = JSON.parse(r.request().postData() || "{}");
      groups = { ...groups, groups: [group(body.members)] };
      saved(body);
      return json(groups);
    }
    if (url.pathname === "/api/providers") return json({ providers: [], presets: [], excluded: [], gateway: { running: true, window: true, url: "http://127.0.0.1:3999" } });
    if (url.pathname.startsWith("/api/")) return json({});
    const file = path.join(assets, url.pathname === "/" ? "index.html" : url.pathname);
    const contentType = { ".html": "text/html", ".js": "text/javascript", ".css": "text/css", ".svg": "image/svg+xml", ".png": "image/png" }[path.extname(file)];
    await r.fulfill({ body: await fs.readFile(file), contentType });
  };
}

for (const engine of (process.env.BROWSER ? [process.env.BROWSER] : ["chromium", "webkit"])) {
  for (const lang of ["en", "zh"]) {
    test(`${engine} ${lang}: the story names the group's models as the group is now`, async (t) => {
      const browser = await (engine === "webkit" ? webkit.launch() : chromium.launch({ channel: process.env.PLAYWRIGHT_CHANNEL || "chromium" }));
      t.after(() => browser.close());
      const page = await (await browser.newContext({ viewport: { width: 1100, height: 760 }, reducedMotion: "reduce" })).newPage();
      page.setDefaultTimeout(5000);
      const errors = [];
      page.on("pageerror", (e) => errors.push(e.message));
      let posted = null;
      const saved = new Promise((res) => { posted = res; });
      await page.route("**/*", serve(lang, (body) => posted(body)));
      await page.goto("http://magpie.test/?view=routing");
      const line = page.locator(".rt-log .rt-steps li").first();
      await line.waitFor();
      // the group now has a/m and c/luna, not the b/m the request was planned
      // with: the models the groups list below names, said as it names them
      assert.match(await line.textContent(), /A · MA · C · Luna$/, "the group's models now");
      assert.doesNotMatch(await line.textContent(), /b\/m/, "a model taken out of the group");
      // removing a model and saving: the story is told again at once
      await page.locator(".rt-group[data-id=hand]").click();
      const remove = page.locator(".rt-gedit button", { hasText: await page.evaluate(() => t("Remove")) });
      await remove.first().waitFor();
      await remove.first().click();
      const save = page.locator(".rt-gedit button", { hasText: await page.evaluate(() => t("Save")) });
      // the editor is taller than the window it is opened in: the reader
      // scrolls down to its Save, as this does
      const view = page.locator("#view-routing");
      const box = await view.boundingBox();
      await page.mouse.move(box.x + box.width / 2, box.y + box.height / 2);
      for (let i = 0; i < 200; i++) {
        const b = await save.boundingBox();
        if (b && b.y + b.height <= box.y + box.height) break;
        if (await view.evaluate((v) => v.scrollTop + v.clientHeight >= v.scrollHeight - 1)) break;
        await page.mouse.wheel(0, 120);
        await page.waitForTimeout(20);
      }
      await save.click();
      const body = await saved;
      assert.deepEqual(body.members, ["c/luna"], "the group saved with the model removed");
      await page.waitForFunction(() => {
        const s = document.querySelector(".rt-log .rt-steps li")?.textContent || "";
        return s.includes("Luna") && !s.includes("MA");
      });
      assert.doesNotMatch(await line.textContent(), /MA/, "the model removed is still named");
      assert.deepEqual(errors, []);
    });
  }
}
