// Run with Node's test runner and Playwright on the module path; see README.md.
// A sign-in's risk step keeps its warning readable and its buttons inside the
// card at 360 and 420 wide, German "Trotzdem anmelden" included (#2099,
// yetone; covers #2824, oxsean). The API is faked here.
const assert = require("node:assert/strict");
const fs = require("node:fs/promises");
const path = require("node:path");
const { test } = require("node:test");
const { chromium, webkit } = require("playwright");

const assets = path.resolve(__dirname, "../assets");

function serve(lang) {
  return async (r) => {
    const url = new URL(r.request().url());
    const json = (data) => r.fulfill({ json: data });
    if (url.pathname === "/boot.js") return r.fulfill({ contentType: "text/javascript", body: `window.bootPrefs = {lang:"${lang}",theme:"light",web:false};` });
    if (url.pathname === "/wails/runtime.js") return r.fulfill({ contentType: "text/javascript", body: "export const Window = {};" });
    if (url.pathname === "/api/state") return json({ agents: [], profiles: [], settings: { lang, theme: "light" } });
    if (url.pathname === "/api/providers") return json({ providers: [{ id: "openai", name: "OpenAI", icon: "openai", preset: "openai", models: [], agents: [], key: { set: true, masked: "sk-…ab12" } }], presets: [], excluded: [], gateway: { running: true, window: true } });
    if (url.pathname === "/api/groups") return json({ groups: [], pools: [] });
    if (url.pathname === "/api/gateway/trace") return json({ routes: [] });
    if (url.pathname.startsWith("/api/signin")) return json({ id: "s1", agent: "claude", state: "waiting", url: "https://claude.ai/oauth/authorize" });
    if (url.pathname.startsWith("/api/")) return json({});
    const file = path.join(assets, url.pathname === "/" ? "index.html" : url.pathname);
    const contentType = { ".html": "text/html", ".js": "text/javascript", ".css": "text/css", ".svg": "image/svg+xml", ".png": "image/png" }[path.extname(file)];
    try { await r.fulfill({ body: await fs.readFile(file), contentType }); } catch { await r.fulfill({ status: 404, body: "" }); }
  };
}

const L = {
  en: { title: "Claude accounts can be suspended", anyway: "Sign in anyway", cancel: "Cancel" },
  zh: { title: "Claude 账号可能被封禁", anyway: "仍然登录", cancel: "取消" },
};

for (const engine of (process.env.BROWSER ? [process.env.BROWSER] : ["chromium", "webkit"])) {
  for (const lang of ["en", "zh", "de"]) {
    for (const width of [360, 420, 1100]) {
      test(`${engine} ${lang} ${width}: the risk warning and its buttons stay inside the card`, async (t) => {
        const browser = await (engine === "webkit" ? webkit.launch() : chromium.launch({ channel: "chromium" }));
        t.after(() => browser.close());
        const page = await (await browser.newContext({ viewport: { width: Math.max(width, 500), height: 900 } })).newPage();
        page.setDefaultTimeout(5000);
        const errors = [];
        page.on("pageerror", (e) => errors.push(e.message));
        await page.route("**/*", serve(lang));
        await page.goto("http://magpie.test/?view=providers");
        await page.locator("#addProvider").click();
        const sheet = page.locator("#addSheet");
        const row = sheet.locator('.tile[data-pick="Claude"]');
        await row.waitFor();
        await row.click();
        const box = sheet.locator(".signing");
        await box.waitFor();
        assert.equal(await box.locator(".mark").textContent(), "!", "the risk step is up");
        const say = (key) => page.evaluate(([lang, key]) => {
          const s = (lang !== "en" && I18N[lang]?.[key]) || key;
          return s;
        }, [lang, key]);
        if (lang === "de") assert.ok((await box.locator(".n").textContent()).length > 0);
        const m = await box.evaluate((b) => {
          const cb = b.getBoundingClientRect();
          const wb = b.querySelector(".tt").getBoundingClientRect();
          const lh = parseFloat(getComputedStyle(b.querySelector(".tt")).lineHeight);
          return {
            cardRight: cb.right, warnW: wb.width, lines: Math.ceil(wb.height / lh),
            over: [...b.querySelectorAll("button")].map((x) => x.getBoundingClientRect().right).filter((r) => r > cb.right + 1),
          };
        });
        assert.ok(m.warnW >= 160, `warning is wide enough (${m.warnW}px)`);
        assert.ok(m.lines <= 5, "the warning stays to a few lines");
        assert.deepEqual(m.over, [], "no button past the card's right edge");
        assert.deepEqual(errors, []);
      });
    }
  }
}
