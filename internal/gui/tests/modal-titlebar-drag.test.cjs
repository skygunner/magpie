// Run with Node's test runner and Playwright on the module path; see README.md.
// #3322 (StudentWeis): in the macOS window, with a context-detail dialog open,
// a press on the titlebar's empty strip could not move the window. The dialog
// makes every child of <body> inert, the titlebar among them, and Wails moves
// the window from the element under the press, an inert one skipped, so the
// press fell through to the page. The titlebar and the sidebar's top strip
// keep their drag now, and the buttons in them are inert instead: a press on
// the empty strip finds a drag region, a press on a tab behind the dialog does
// nothing, the dialog keeps the focus, and closing it gives the buttons back.
// Chromium and WebKit; no backend, the API is faked here.
const assert = require("node:assert/strict");
const fs = require("node:fs/promises");
const path = require("node:path");
const { test } = require("node:test");
const { chromium, webkit } = require("playwright");

const assets = path.resolve(__dirname, "../assets");
const volc = {
  id: "volc", name: "doubao-seed", icon: "volcengine-color", chat: "https://ark.example.com/api/v3", responses: "", anthropic: "", catalog: "", host: "ark.example.com",
  models: [{ id: "doubao-seed", name: "doubao-seed", on: true }], fallback: [], headers: {},
  agents: [], key: { set: true, masked: "sk-…one" }, keyList: [], balanceToken: { takes: false, set: false }, proxy: "", ready: true,
};

function serve() {
  return async (route) => {
    const url = new URL(route.request().url());
    const json = (data) => route.fulfill({ json: data });
    if (url.pathname === "/boot.js") return route.fulfill({ contentType: "text/javascript", body: `window.bootPrefs = {lang:"en",theme:"light",web:false};` });
    if (url.pathname === "/wails/runtime.js") return route.fulfill({ contentType: "text/javascript", body: "export const Window = {};" });
    if (url.pathname === "/api/state") return json({ agents: [], profiles: [], settings: { lang: "en", theme: "light" } });
    if (url.pathname === "/api/providers") return json({ providers: [volc], presets: [], excluded: [], gateway: { running: true, window: true } });
    if (url.pathname === "/api/groups") return json({ groups: [] });
    if (url.pathname === "/api/usage/quotas") return json([]);
    if (url.pathname.startsWith("/api/")) return json({});
    const file = path.join(assets, url.pathname === "/" ? "index.html" : url.pathname);
    const contentType = { ".html": "text/html", ".js": "text/javascript", ".css": "text/css", ".svg": "image/svg+xml", ".png": "image/png" }[path.extname(file)];
    try { await route.fulfill({ body: await fs.readFile(file), contentType }); } catch { await route.fulfill({ status: 404, body: "" }); }
  };
}

// the widest run of the strip, in px, where the element under the press drags
// the window: Wails reads --wails-draggable off that element (drag.ts)
const dragRun = (page) => page.evaluate(() => {
  const drag = (node) => !!node && getComputedStyle(node).getPropertyValue("--wails-draggable").trim() === "drag";
  const widest = (box, y) => {
    let best = 0, run = 0;
    for (let x = Math.ceil(box.left) + 2; x < box.right - 2; x += 2) {
      if (drag(document.elementFromPoint(x, y + box.height / 2))) { run += 2; best = Math.max(best, run); } else run = 0;
    }
    return best;
  };
  const header = document.querySelector("header.top").getBoundingClientRect();
  const side = document.querySelector("aside.sidebar");
  const top = side && !side.hidden ? side.querySelector(".side-top").getBoundingClientRect() : null;
  return { header: widest(header, header.top), sideTop: top && widest(top, top.top) };
});

// a press on the middle of a control in the titlebar, as a person's: the
// control's own click must not fire while the dialog is up, and the dialog
// must still be there afterwards (a press on the scrim would close it)
async function press(page, selector, note) {
  const box = await page.locator(selector).boundingBox();
  assert(box, `${note}: the control is on the page`);
  const top = await page.evaluate(() => document.querySelector("#modal").getBoundingClientRect().top);
  assert(box.y + box.height <= top, `${note}: it is over the dialog, not under its scrim (${box.y + box.height} > ${top})`);
  await page.evaluate((sel) => {
    document.querySelector(sel).addEventListener("click", () => { document.body.dataset.behindClicked = "yes"; }, { capture: true });
  }, selector);
  await page.mouse.click(box.x + box.width / 2, box.y + box.height / 2);
  assert.equal(await page.evaluate(() => document.body.dataset.behindClicked || ""), "", `${note}: the dialog keeps the press`);
  assert.equal(await page.locator("#modal").evaluate((modal) => modal.hidden || modal.classList.contains("out")), false, `${note}: the dialog is still open`);
}

for (const engine of (process.env.BROWSER ? [process.env.BROWSER] : ["chromium", "webkit"])) {
  for (const layout of ["side", "top"]) {
    test(`${engine}: a dialog leaves the window's titlebar draggable (${layout} navigation)`, async (t) => {
      const browser = await (engine === "webkit" ? webkit.launch() : chromium.launch({ channel: "chromium" }));
      const errors = [];
      t.after(async () => {
        if (errors.length) console.log(errors);
        await browser.close();
      });
      const ctx = await browser.newContext({ viewport: { width: 1100, height: 760 }, reducedMotion: "reduce" });
      await ctx.addInitScript((picked) => {
        Object.defineProperty(Navigator.prototype, "platform", { get: () => "MacIntel" });
        localStorage.setItem("magpie.navLayout", picked);
      }, layout);
      const page = await ctx.newPage();
      page.setDefaultTimeout(5000);
      page.on("pageerror", (e) => errors.push(e.message));
      await page.route("**/*", serve());
      await page.goto("http://magpie.test/?mode=window&view=providers");
      await page.locator("#nav button.on").waitFor();
      assert.equal(await page.evaluate(() => document.body.classList.contains("side-nav")), layout === "side", "the picked navigation is shown");

      const row = page.locator('.row.provider[data-id="volc"]');
      await row.click();
      await page.getByRole("dialog").waitFor();

      const open = await dragRun(page);
      assert(open.header >= 60, `the titlebar's empty strip still moves the window: ${open.header}px`);
      if (layout === "side") assert(open.sideTop >= 60, `the sidebar's top strip still moves the window: ${open.sideTop}px`);

      // the controls behind the dialog are dead: the titlebar's own above the
      // dialog, and the sidebar's under its scrim, where Tab cannot reach them
      const navInert = await page.evaluate(() => [...document.querySelectorAll("#nav button")].every((button) => !!button.closest("[inert]")));
      assert.equal(navInert, true, "the tabs under the dialog are inert");
      if (layout === "side") {
        const was = await page.evaluate(() => document.body.className);
        await press(page, "#sideToggle", "the sidebar's toggle in the titlebar");
        assert.equal(await page.evaluate(() => document.body.className), was, "a press on the toggle does not lay the page out again");
      } else {
        await press(page, '#nav button[data-view="providers"]', "a tab in the titlebar");
      }

      // the dialog keeps the focus through a full turn of its own controls
      await page.locator("#modal .bar button:last-child").focus();
      for (let i = 0; i < 12; i++) {
        await page.keyboard.press("Tab");
        assert.equal(await page.evaluate(() => !!document.activeElement.closest("#modal")), true, `Tab ${i + 1} stays in the dialog`);
      }

      await page.keyboard.press("Escape");
      await page.locator("#modal").waitFor({ state: "hidden" });
      const closed = await dragRun(page);
      assert(closed.header >= 60, "the titlebar still moves the window after the dialog");
      await page.locator('#nav button[data-view="providers"]').click();
      assert.equal(await page.locator("#view-providers").evaluate((view) => !view.hidden), true, "the tabs take clicks again");
      assert.deepEqual(errors, []);
      await ctx.close();
    });
  }

  // Settings, the Library and Plugins put their tool bar in the titlebar strip
  // in this layout, so that strip is the one a press must find there
  test(`${engine}: a page's tool bar in the titlebar still drags under a dialog`, async (t) => {
    const browser = await (engine === "webkit" ? webkit.launch() : chromium.launch({ channel: "chromium" }));
    const errors = [];
    t.after(async () => {
      if (errors.length) console.log(errors);
      await browser.close();
    });
    const ctx = await browser.newContext({ viewport: { width: 1100, height: 760 }, reducedMotion: "reduce" });
    await ctx.addInitScript(() => {
      Object.defineProperty(Navigator.prototype, "platform", { get: () => "MacIntel" });
      localStorage.setItem("magpie.navLayout", "side");
    });
    const page = await ctx.newPage();
    page.setDefaultTimeout(5000);
    page.on("pageerror", (e) => errors.push(e.message));
    await page.route("**/*", serve());
    await page.goto("http://magpie.test/?mode=window");
    await page.locator("#nav button.on").waitFor();
    await page.locator("#prefs").click();
    const head = page.locator("#view-settings .set-head");
    await head.waitFor();

    await page.evaluate(() => {
      const content = document.createElement("div");
      content.className = "editor";
      content.innerHTML = '<div class="ehead"><b>Fixture</b></div><div class="ebody"><label for="fixture">Name</label><input id="fixture"></div><div class="bar"><button type="button">Cancel</button></div>';
      openModal(content);
    });
    await page.getByRole("dialog").waitFor();
    const strip = await page.evaluate(() => {
      const drag = (node) => !!node && getComputedStyle(node).getPropertyValue("--wails-draggable").trim() === "drag";
      const box = document.querySelector("#view-settings .set-head").getBoundingClientRect();
      let best = 0, run = 0;
      for (let x = Math.ceil(box.left) + 2; x < box.right - 2; x += 2) {
        if (drag(document.elementFromPoint(x, box.top + box.height / 2))) { run += 2; best = Math.max(best, run); } else run = 0;
      }
      return { run: best, controls: [...document.querySelectorAll("#view-settings .set-head button, #view-settings .set-head input")].every((control) => control.inert),
        page: [...document.querySelector("#view-settings .set-head").parentElement.children].filter((child) => !child.classList.contains("set-head")).every((child) => child.inert) };
    });
    assert(strip.run >= 60, `the tool bar in the titlebar still moves the window: ${strip.run}px`);
    assert.equal(strip.controls, true, "the tool bar's own controls are inert");
    assert.equal(strip.page, true, "the page under the dialog stays inert");
    await page.evaluate(() => closeModal());
    await page.locator("#modal").waitFor({ state: "hidden" });
    assert.equal(await head.evaluate((element) => element.querySelector("input, button")?.inert ?? false), false, "the tool bar takes presses again");
    assert.deepEqual(errors, []);
    await ctx.close();
  });
}
