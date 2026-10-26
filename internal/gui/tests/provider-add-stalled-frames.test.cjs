// Run with Node's test runner and Playwright on the module path; see README.md.
// #3634: the opening click on #addProvider at 420px waited for the button's box
// over two animation frames, and on a loaded machine WebKit draws no frames for
// seconds while the page's own tasks run on, so it sat out the 5s page default
// (the stall openPart's tab sat out, #2150). The sheet now opens with Enter,
// which needs no frame to wait for. Reproduced the way 62e012b3 did, by
// duty-cycling WebKit's UI process (SIGSTOP 990ms / SIGCONT 10ms, the frames
// come from it) around the opening and asking for the sheet: a WebKit that
// draws almost no frames still opens it. The harness finds that process in
// /proc, so the stall runs on Linux; elsewhere the subtest is skipped and the
// sheet still opens there, unstalled.
const assert = require("node:assert/strict");
const fs = require("node:fs");
const fsPromises = require("node:fs/promises");
const path = require("node:path");
const { execSync } = require("node:child_process");
const { test } = require("node:test");
const { webkit } = require("playwright");
const { openAddSheet } = require("./reader.cjs");

const assets = path.resolve(__dirname, "../assets");

function serve() {
  const providers = { providers: [{ id: "anthropic", name: "Anthropic", icon: "anthropic", preset: "anthropic", models: [], agents: [], key: { set: true, masked: "sk-…ab12" } }], presets: [], excluded: [], gateway: { running: true, window: true } };
  return async (route) => {
    const url = new URL(route.request().url());
    const json = (data) => route.fulfill({ json: data });
    if (url.pathname === "/boot.js") return route.fulfill({ contentType: "text/javascript", body: 'window.bootPrefs = {lang:"en",theme:"light",web:false};' });
    if (url.pathname === "/wails/runtime.js") return route.fulfill({ contentType: "text/javascript", body: "export const Window = {};" });
    if (url.pathname === "/api/state") return json({ agents: [], profiles: [], settings: { lang: "en", theme: "light" } });
    if (url.pathname === "/api/providers") return json(providers);
    if (url.pathname === "/api/groups") return json({ groups: [], pools: [] });
    if (url.pathname.startsWith("/api/")) return json({});
    const file = path.join(assets, url.pathname === "/" ? "index.html" : url.pathname);
    try { await route.fulfill({ body: await fsPromises.readFile(file), contentType: { ".html": "text/html", ".js": "text/javascript", ".css": "text/css", ".svg": "image/svg+xml", ".png": "image/png" }[path.extname(file)] }); } catch { await route.fulfill({ status: 404, body: "" }); }
  };
}

// WebKit's frames come from its UI process: stopping it starves the page's
// frames while the page's own tasks run on. It is a descendant of this test.
const kidsOf = (pid) => {
  const all = new Set();
  let tids = [pid];
  try { tids = fs.readdirSync("/proc/" + pid + "/task").map(Number); } catch {}
  for (const t of tids) {
    try { for (const c of fs.readFileSync("/proc/" + pid + "/task/" + t + "/children", "utf8").trim().split(/\s+/).filter(Boolean)) all.add(Number(c)); } catch {}
  }
  return [...all];
};
const descendants = (pid, out = []) => { for (const c of kidsOf(pid)) { out.push(c); descendants(c, out); } return out; };
const nameOf = (pid) => { try { return fs.readFileSync("/proc/" + pid + "/cmdline", "utf8").split("\0")[0].split("/").pop(); } catch { return ""; } };

test("webkit 420px: the add sheet opens with WebKit drawing almost no frames", async (t) => {
  if (process.platform !== "linux") return t.skip("the stall harness reads /proc");
  const browser = await webkit.launch();
  const stopped = [];
  t.after(async () => {
    // a browser left stopped can't be closed by its own process
    for (const pid of stopped) { try { execSync("kill -CONT " + pid); } catch {} }
    await browser.close();
  });
  const page = await (await browser.newContext({ viewport: { width: 420, height: 800 }, reducedMotion: "reduce" })).newPage();
  page.setDefaultTimeout(5000);
  const errors = [];
  page.on("pageerror", (e) => errors.push(e.message));
  await page.route("**/*", serve());
  await page.goto("http://magpie.test/?view=providers");
  const ui = descendants(process.pid).find((p) => nameOf(p) === "MiniBrowser");
  assert(ui, "the browser's UI process is under this test");
  stopped.push(ui);
  // 10ms of frames a second, from before the opening to after it
  let stop = false;
  (function duty() {
    if (stop) { try { execSync("kill -CONT " + ui + " 2>/dev/null || true"); } catch {} return; }
    try { execSync("kill -STOP " + ui + " 2>/dev/null || true"); } catch {}
    setTimeout(() => { try { execSync("kill -CONT " + ui + " 2>/dev/null || true"); } catch {}; setTimeout(duty, 10); }, 990);
  })();
  t.after(() => { stop = true; });
  await openAddSheet(page);
  stop = true;
  // read in the page, not through a wait: a wait polls on frames, and there
  // are none to poll with
  const opened = await page.evaluate(() => {
    const sheet = document.querySelector("#addSheet");
    return !!sheet && !sheet.hidden && !!sheet.querySelector(".custom-foot .custom");
  });
  assert(opened, "the add sheet opened with the custom entry in it");
  assert.deepEqual(errors, []);
});
