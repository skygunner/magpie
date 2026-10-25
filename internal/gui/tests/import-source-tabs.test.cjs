const assert = require("node:assert/strict");
const fs = require("node:fs/promises");
const path = require("node:path");
const { test } = require("node:test");
const { chromium, webkit } = require("playwright");
const assets = process.env.ASSET_DIR || path.resolve(__dirname, "../assets");
const engines = process.env.BROWSER ? [process.env.BROWSER] : ["chromium", "webkit"];

const sources = () => [
  { id: "alma", name: "Alma", path: "/fixture/alma", found: false, items: [] },
  { id: "cc-switch", name: "CC Switch", path: "/fixture/cc-switch", found: true, items: [
    { ref: "relay", from: "config.json", fingerprint: "cc1", status: "new",
      provider: { id: "relay", name: "Relay", chat: "https://relay.example/v1", key: "sk-…1234", models: ["fixture-model"] } },
  ] },
  { id: "claude-code", name: "Claude Code", path: "/fixture/claude", found: false, items: [] },
  { id: "codex", name: "Codex", path: "/fixture/codex", found: true, items: [
    { ref: "oa", from: "config.toml", fingerprint: "cx1", status: "new",
      provider: { id: "oa", name: "OA", responses: "https://oa.example/v1", key: "sk-…1234" } },
  ] },
  { id: "opencode", name: "OpenCode", path: "/fixture/opencode", found: true, items: [
    { ref: "go", from: "auth.json", fingerprint: "oc1", status: "new",
      provider: { id: "go", name: "Go", chat: "https://go.example/v1", key: "sk-…1234" } },
  ] },
];

for (const engine of engines) {
  for (const lang of ["en", "zh"]) {
    for (const width of [390, 900]) {
      test(`${engine} ${lang} ${width}px: every import source tile fits its row`, async (t) => {
        const browser = await (engine === "webkit" ? webkit.launch() : chromium.launch({ channel: "chromium" }));
        const page = await browser.newPage({ viewport: { width, height: 900 }, reducedMotion: "reduce" });
        page.setDefaultTimeout(7000);
        const errors = [];
        page.on("pageerror", (e) => errors.push(e.stack || e.message));
        await page.route("**/*", async (route) => {
          const req = route.request(), url = new URL(req.url());
          const json = (data) => route.fulfill({ json: data });
          if (url.pathname === "/boot.js") return route.fulfill({ contentType: "text/javascript", body: `window.bootPrefs={lang:"${lang}",theme:"light",web:true};` });
          if (url.pathname === "/api/state") return json({ agents: [], profiles: [], settings: { lang, theme: "light" } });
          if (url.pathname === "/api/providers") return json({ providers: [], presets: [], excluded: [], gateway: { running: true, window: true } });
          if (url.pathname === "/api/groups") return json({ groups: [], pools: [] });
          if (url.pathname === "/api/importapps/discovery") return json([]);
          if (url.pathname === "/api/importapps") return json(sources());
          if (url.pathname === "/api/gateway/trace") return new Promise(() => {});
          if (url.pathname.startsWith("/api/")) return json({});
          const file = path.join(assets, url.pathname === "/" ? "index.html" : url.pathname);
          const contentType = { ".html": "text/html", ".js": "text/javascript", ".css": "text/css", ".svg": "image/svg+xml", ".png": "image/png" }[path.extname(file)];
          try { await route.fulfill({ body: await fs.readFile(file), contentType }); } catch { await route.fulfill({ status: 404, body: "" }); }
        });
        t.after(async () => {
          await browser.close();
          assert.deepEqual(errors, []);
        });
        await page.goto("http://magpie.test/?view=providers");
        await page.waitForFunction(() => typeof openImportApps === "function");
        await page.evaluate(() => openImportApps());
        const tabs = page.locator("#modal .importapps .apptab");
        await tabs.first().waitFor();
        assert.equal(await tabs.count(), 5);
        const row = await page.evaluate(() => {
          const sc = document.querySelector("#modal .importapps .apptabs");
          const r = sc.getBoundingClientRect();
          return {
            width: r.width,
            overflow: sc.scrollWidth - sc.clientWidth,
            tiles: [...sc.querySelectorAll(".apptab")].map((b) => {
              const q = b.getBoundingClientRect();
              return { name: b.textContent, left: q.left - r.left, right: q.right - r.left, width: q.width };
            }),
          };
        });
        for (const tile of row.tiles) {
          assert.ok(tile.width > 0, `${tile.name} has no width`);
          assert.ok(tile.left >= -0.5 && tile.right <= row.width + 0.5,
            `${tile.name} is cut off: ${tile.left.toFixed(1)}..${tile.right.toFixed(1)} of ${row.width.toFixed(1)}`);
        }
        assert.ok(row.overflow <= 1, `the row hides ${row.overflow}px of its tiles`);
      });
    }
  }
}
