// Run with Node's test runner and Playwright on the module path; see README.md.
// With prefers-reduced-motion, the decorative loops stop instead of keeping a
// timeline of .01ms iterations (the global fallback makes each iteration
// instant, an infinite one still never ends): the model chip's waiting dot,
// the library icon's bob, the routing live dot's pulse (its own stop was
// beaten by the .on rule's specificity) and the pulse before a provider's
// tag. A status loop keeps animating: a waiting request's dot.
const assert = require("node:assert/strict");
const fs = require("node:fs/promises");
const path = require("node:path");
const { test } = require("node:test");
const { chromium, webkit } = require("playwright");

const assets = path.resolve(__dirname, "../assets");

async function serve(route) {
  const url = new URL(route.request().url());
  const json = (data) => route.fulfill({ json: data });
  if (url.pathname === "/boot.js") return route.fulfill({ contentType: "text/javascript", body: 'window.bootPrefs = {lang:"en",theme:"light",web:true};' });
  if (url.pathname === "/wails/runtime.js") return route.fulfill({ contentType: "text/javascript", body: "export const Window = {};" });
  if (url.pathname.startsWith("/api/")) return json({});
  const file = path.join(assets, url.pathname === "/" ? "index.html" : url.pathname);
  const contentType = { ".html": "text/html", ".js": "text/javascript", ".css": "text/css", ".svg": "image/svg+xml", ".png": "image/png" }[path.extname(file)];
  await route.fulfill({ body: await fs.readFile(file).catch(() => ""), contentType });
}

for (const engine of (process.env.BROWSER ? [process.env.BROWSER] : ["chromium", "webkit"])) {
  test(engine + ": reduced motion stops the decorative loops, keeps the status ones", async (t) => {
    const browser = await (engine === "webkit" ? webkit.launch() : chromium.launch({ channel: "chromium" }));
    t.after(() => browser.close());
    const context = await browser.newContext({ viewport: { width: 900, height: 560 }, reducedMotion: "reduce" });
    const page = await context.newPage();
    page.setDefaultTimeout(5000);
    const errors = [];
    page.on("pageerror", (e) => errors.push(e.message));
    await page.route("**/*", serve);
    await page.goto("http://magpie.test/");
    await page.locator(".side-bar").first().waitFor({ state: "attached" });
    const names = await page.evaluate(() => {
      const chip = document.createElement("span");
      chip.className = "mchip";
      chip.innerHTML = '<i class="tdot wait"></i>';
      const lib = document.createElement("span");
      lib.className = "lib-icon busy";
      lib.innerHTML = '<svg viewBox="0 0 1 1"></svg>';
      const live = document.createElement("i");
      live.className = "rt-livedot on";
      const wait = document.createElement("div");
      wait.className = "rt-req wait";
      wait.innerHTML = '<span class="to"><i></i></span>';
      const tag = document.createElement("span");
      tag.className = "rp-tag";
      document.body.append(chip, lib, live, wait, tag);
      const name = (el, pseudo) => getComputedStyle(el, pseudo).animationName;
      return {
        chip: name(chip.querySelector(".tdot.wait")),
        lib: name(lib.querySelector("svg")),
        live: name(live),
        wait: name(wait.querySelector(".to i")),
        tag: name(tag, "::before"),
      };
    });
    assert.deepEqual(names, { chip: "none", lib: "none", live: "none", wait: "rtlive", tag: "none" });
    assert.deepEqual(errors, []);
  });
}
