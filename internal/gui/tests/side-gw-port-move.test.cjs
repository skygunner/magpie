// Run with Node's test runner and Playwright on the module path; see README.md.
// #3531 (yetone): moving the gateway from Settings › Network › Gateway
// address › Apply left the sidebar's gateway line saying the old port until
// a reload — the line reads the providers, and nothing told it the gateway
// had moved. The save's answer carries the new address; the line is drawn
// from it now. Both engines, English and Chinese; no backend, the API is
// faked here.
const assert = require("node:assert/strict");
const fs = require("node:fs/promises");
const path = require("node:path");
const { test } = require("node:test");
const { chromium, webkit } = require("playwright");

const assets = path.resolve(__dirname, "../assets");
const settings = (port) => ({ lang: "en", theme: "light", port, gateway: "http://127.0.0.1:" + port });

function serve(lang, port) {
  const st = { ...settings(port), lang };
  const state = { agents: [], profiles: [], settings: st };
  return async (r) => {
    const url = new URL(r.request().url());
    const json = (data) => r.fulfill({ json: data });
    if (url.pathname === "/boot.js") return r.fulfill({ contentType: "text/javascript", body: `window.bootPrefs = {lang:"${lang}",theme:"light",web:false};` });
    if (url.pathname === "/wails/runtime.js") return r.fulfill({ contentType: "text/javascript", body: "export const Window = {};" });
    if (url.pathname === "/api/state") return json(state);
    if (url.pathname === "/api/settings") return json(st);
    if (url.pathname === "/api/providers") return json({ providers: [], presets: [], excluded: [], gateway: { url: "http://127.0.0.1:" + port, running: true } });
    if (url.pathname === "/api/settings/port") {
      const body = JSON.parse(r.request().postData() || "{}");
      const next = settings(body.port);
      Object.assign(st, next);
      return json({ port: { moved: [] }, settings: next });
    }
    if (url.pathname === "/api/gateway/trace") return url.searchParams.get("wait") ? new Promise(() => {}) : json({ mine: true, now: new Date().toISOString(), seq: 0, totals: {}, routes: [] });
    if (url.pathname === "/api/gateway/history") return json({ cut: false, days: [], routes: [] });
    if (url.pathname === "/api/groups") return json({ groups: [], models: [], pools: [] });
    if (url.pathname === "/api/plugins") return json({ plugins: [] });
    if (url.pathname.startsWith("/api/")) return json({});
    const file = path.join(assets, url.pathname === "/" ? "index.html" : url.pathname);
    const contentType = { ".html": "text/html", ".js": "text/javascript", ".css": "text/css", ".svg": "image/svg+xml", ".png": "image/png" }[path.extname(file)];
    try { await r.fulfill({ body: await fs.readFile(file), contentType }); } catch { await r.fulfill({ status: 404, body: "" }); }
  };
}

for (const engine of (process.env.BROWSER ? [process.env.BROWSER] : ["chromium", "webkit"])) {
  for (const lang of ["en", "zh"]) {
    test(`${engine} ${lang}: the sidebar's gateway line moves with the port (#3531)`, async () => {
      const browser = await { chromium, webkit }[engine].launch();
      try {
        const page = await (await browser.newContext({ viewport: { width: 1024, height: 760 }, reducedMotion: "reduce" })).newPage();
        page.setDefaultTimeout(5000);
        await page.route("**/*", serve(lang, 3483));
        await page.goto("http://magpie.test/?view=settings&tab=network");
        const line = page.locator("#sideGw span");
        await line.waitFor();
        assert.ok((await line.textContent()).includes("3483"), await line.textContent());
        await page.locator(".port-row").waitFor();
        await page.locator("input.gateway-port").fill("3486");
        await page.locator(".port-row button").click();
        await page.waitForTimeout(300);
        const now = await line.textContent();
        assert.ok(now.includes("3486"), `the sidebar kept the old port: "${now}"`);
        assert.ok(!now.includes("3483"), `the sidebar kept the old port: "${now}"`);
        await page.close();
      } finally {
        await browser.close();
      }
    });
  }
}
