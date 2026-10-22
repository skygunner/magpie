// Run with Node's test runner and Playwright on the module path; see README.md.
// A custom provider's Base URL takes a URL for each API in turn (#1231:
// 不同协议的地址现在只能保存当前的，不能一下保存多个). A URL typed and not
// saved still follows to an API with none when that API is picked after it,
// spelled as that API wants it (#73, #105). Typed over there, the new URL is
// that API's own and the first goes back to the API it was typed for: both
// are saved. Left as it was carried, it is a move, as before. For a new
// provider and a saved one, in all five languages, Chromium and WebKit, at
// a narrow window too. A round trip through Gemini retains the original
// spelling instead of carrying its /v1beta back to Anthropic (#3233).
const assert = require("node:assert/strict");
const fs = require("node:fs/promises");
const path = require("node:path");
const { test } = require("node:test");
const { chromium, webkit } = require("playwright");

const assets = path.resolve(__dirname, "../assets");
const ROOT = "https://apihub.relay.example.com";

function serve(lang, posts) {
  const state = { agents: [], profiles: [], settings: { lang, theme: "light" } };
  const base = { icon: "generic", responses: "", anthropic: "", models: [{ id: "m", name: "m", on: true }], agents: [], fallback: [], headers: {}, key: { set: true, masked: "sk-…1234" }, keyList: [], ready: true };
  const providers = { providers: [{ ...base, id: "plain", name: "Plain", chat: ROOT + "/v1" }], presets: [], excluded: [], gateway: { running: true, window: true } };
  return async (route) => {
    const url = new URL(route.request().url());
    const json = (data) => route.fulfill({ json: data });
    if (url.pathname === "/boot.js") return route.fulfill({ contentType: "text/javascript", body: `window.bootPrefs = {lang:${JSON.stringify(lang)},theme:"light",web:true};` });
    if (url.pathname === "/wails/runtime.js") return route.fulfill({ contentType: "text/javascript", body: "export const Window = {};" });
    if (url.pathname === "/api/state") return json(state);
    if (url.pathname === "/api/providers") return json(providers);
    if (url.pathname === "/api/groups") return json({ groups: [] });
    if (url.pathname.startsWith("/api/provider/")) {
      const body = route.request().postDataJSON();
      posts.push({ path: url.pathname, body });
      if (url.pathname === "/api/provider/detect") return json({ results: [
        { protocol: "chat", base: ROOT + "/v1", ok: true, status: 200 },
        { protocol: "responses", base: ROOT + "/v1", ok: true, status: 200 },
        { protocol: "anthropic", base: ROOT, ok: true, status: 200 },
        { protocol: "gemini", base: ROOT + "/v1beta", ok: true, status: 200 },
      ] });
      if (url.pathname === "/api/provider/save") {
        let saved = providers.providers.find((p) => p.id === body.id);
        if (!saved) providers.providers.push(saved = { ...base });
        for (const key of ["id", "name", "chat", "responses", "anthropic", "gemini", "decide", "baseAPI"]) saved[key] = body[key];
      }
      return json(providers);
    }
    if (url.pathname.startsWith("/api/")) return json({});
    const file = path.join(assets, url.pathname === "/" ? "index.html" : url.pathname);
    await route.fulfill({ body: await fs.readFile(file), contentType: { ".html": "text/html", ".js": "text/javascript", ".css": "text/css", ".svg": "image/svg+xml", ".png": "image/png" }[path.extname(file)] });
  };
}

for (const engine of (process.env.BROWSER ? [process.env.BROWSER] : ["chromium", "webkit"])) {
  for (const [lang, width] of ["en", "zh", "zh-TW", "ja", "de"].flatMap((lang) => [900, 440].map((width) => [lang, width]))) {
    test(`${engine} ${lang} ${width}px: a Base URL for each API, typed in turn, is saved`, async (t) => {
      const browser = await (engine === "webkit" ? webkit.launch() : chromium.launch({ channel: process.env.PLAYWRIGHT_CHANNEL || "chromium" }));
      const page = await (await browser.newContext({ viewport: { width, height: 760 }, reducedMotion: "reduce" })).newPage();
      t.after(() => browser.close());
      page.setDefaultTimeout(5000);
      const errors = [], posts = [];
      page.on("pageerror", (e) => errors.push(e.message));
      await page.route("**/*", serve(lang, posts));
      const [add, save] = { en: ["Add", "Save"], zh: ["添加", "保存"], "zh-TW": ["新增", "儲存"], ja: ["追加", "保存"], de: ["Hinzufügen", "Speichern"] }[lang];
      const L = { add, save };
      const ed = page.locator(".editor");
      const baseURL = ed.locator(".base-url");
      const api = (v) => ed.locator(`.segs .opt[data-api="${v}"]`).click();
      const more = () => ed.locator("details.more input[type=url]").evaluateAll((is) => is.map((i) => i.value));
      const sent = async (n) => {
        for (let i = 0; i < 60 && posts.filter((p) => p.path === "/api/provider/save").length < n; i++) await page.waitForTimeout(50);
        const all = posts.filter((p) => p.path === "/api/provider/save");
        assert.equal(all.length, n, "saved");
        return all[n - 1].body;
      };
      const addNew = async (name) => {
        await page.goto("http://magpie.test/?view=providers");
        await page.locator("#addProvider").click();
        await page.locator("#addSheet .custom-foot .custom").click();
        await baseURL.waitFor();
        await ed.locator('input[placeholder]').first().fill(name);
      };
      const press = (label) => ed.locator(":scope > .bar").getByRole("button", { name: label, exact: true }).click();

      // new: OpenAI's URL, then Anthropic picked, its own URL typed there
      await addNew("Relay");
      await baseURL.fill(ROOT + "/v1");
      await api("anthropic");
      assert.equal(await baseURL.inputValue(), ROOT, "carried over, spelled for Anthropic (#105)");
      await baseURL.fill(ROOT + "/claude");
      assert((await more()).includes(ROOT + "/v1"), "OpenAI's URL is back under More endpoints: " + (await more()));
      await press(L.add);
      let body = await sent(1);
      assert.equal(body.chat, ROOT + "/v1", "the OpenAI URL is saved too (#1231)");
      assert.equal(body.anthropic, ROOT + "/claude");
      assert.equal(body.baseAPI, "anthropic");

      // new: the API picked after the URL, nothing typed there: still a move (#73)
      await addNew("Responses only");
      await baseURL.fill(ROOT + "/v1");
      await api("responses");
      assert.equal(await baseURL.inputValue(), ROOT + "/v1");
      await press(L.add);
      body = await sent(2);
      assert.equal(body.responses, ROOT + "/v1");
      assert.equal(body.chat || "", "", "the URL moved to Responses (#73)");
      assert.equal(body.baseAPI, "responses");

      // saved: a new chat URL typed, then Anthropic's typed after it
      await page.goto("http://magpie.test/?view=providers");
      await page.locator('#providers .row[data-id="plain"]').click();
      await baseURL.waitFor();
      await baseURL.fill(ROOT + "/v2");
      await api("anthropic");
      await baseURL.fill(ROOT + "/claude");
      await api("openai");
      assert.equal(await baseURL.inputValue(), ROOT + "/v2", "OpenAI's tab shows its new URL");
      await press(L.save);
      body = await sent(3);
      assert.equal(body.chat, ROOT + "/v2");
      assert.equal(body.anthropic, ROOT + "/claude");
      // A URL carried through several APIs is always converted from the
      // original spelling. Returning to its original tab restores it exactly.
      await addNew("Round trip");
      await api("anthropic");
      await baseURL.fill(ROOT + "/v1");
      for (const protocol of ["gemini", "responses", "openai", "decide", "anthropic"]) await api(protocol);
      assert.equal(await baseURL.inputValue(), ROOT + "/v1", "Anthropic round trip must not inherit Gemini's /v1beta (#3233)");
      await press(L.add);
      body = await sent(4);
      assert.equal(body.anthropic, ROOT + "/v1");
      for (const key of ["chat", "responses", "gemini", "decide"]) assert.equal(body[key] || "", "", key + " was only visited");
      await page.goto("http://magpie.test/?view=providers");
      await page.locator(`#providers .row[data-id="${body.id}"]`).click();
      await baseURL.waitFor();
      assert.equal(await baseURL.inputValue(), ROOT + "/v1", "reopening uses the saved Anthropic spelling");
      for (const protocol of ["gemini", "responses", "openai", "anthropic"]) await api(protocol);
      assert.equal(await baseURL.inputValue(), ROOT + "/v1", "saved URL stays on its own protocol");

      // Typing over the carried URL after more than one switch restores
      // the first input, rather than the last protocol's converted spelling.
      await addNew("Two endpoints");
      await api("anthropic");
      await baseURL.fill(ROOT + "/claude");
      await api("gemini");
      await api("responses");
      await baseURL.fill(ROOT + "/responses");
      await api("anthropic");
      assert.equal(await baseURL.inputValue(), ROOT + "/claude", "typing at the destination restores the original protocol (#1231)");
      await press(L.add);
      body = await sent(5);
      assert.equal(body.anthropic, ROOT + "/claude");
      assert.equal(body.responses, ROOT + "/responses");
      assert.equal(body.gemini || "", "");
      // Detection supplies independent, normalized addresses. Visiting tabs
      // after taking them must not change the Anthropic or Gemini endpoint.
      await page.goto("http://magpie.test/?view=providers");
      await page.locator('#providers .row[data-id="plain"]').click();
      await baseURL.waitFor();
      await api("anthropic");
      await baseURL.fill(ROOT + "/v1");
      await ed.locator(".detect-row > button").first().click();
      await ed.locator(".detect-acts button").first().click();
      for (const protocol of ["gemini", "responses", "openai", "anthropic"]) await api(protocol);
      assert.equal(await baseURL.inputValue(), ROOT);
      await press(L.save);
      body = await sent(6);
      assert.equal(body.anthropic, ROOT);
      assert.equal(body.gemini, ROOT + "/v1beta");
      assert.deepEqual(errors, []);
    });
  }
}
