const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const vm = require("node:vm");
const { test } = require("node:test");
const { setImmediate } = require("node:timers/promises");

test("live logo fixture holds trace updates until the story is checked", async () => {
  const source = fs.readFileSync(path.join(__dirname, "routing-log-icons.test.cjs"), "utf8");
  const start = source.indexOf("function serveLive(");
  const end = source.indexOf("\nfor (const engine", start);
  // Expire every old timer immediately: a slow reader must still own progress.
  const serveLive = vm.runInNewContext(source.slice(start, end) + "\nserveLive", {
    URL, serve: () => () => assert.fail("unexpected non-trace request"),
    setTimeout: (fn) => queueMicrotask(fn),
  });
  const trace = serveLive("en", [{ id: 1 }, { id: 2 }, { id: 3 }]);
  const seen = [];
  const request = () => trace({
    request: () => ({ url: () => "http://magpie.test/api/gateway/trace?wait=1" }),
    fulfill: ({ json }) => { seen.push(json.routes[0].id); },
  });
  await request();
  assert.deepEqual(seen, [1]);
  for (const id of [2, 3]) {
    const pending = request();
    await setImmediate();
    assert.deepEqual(seen, Array.from({ length: id - 1 }, (_, i) => i + 1), "trace advanced before the story was checked");
    trace.next();
    await pending;
    assert.equal(seen.at(-1), id);
  }
  const pending = request();
  await setImmediate();
  assert.deepEqual(seen, [1, 2, 3], "the final long poll must stay pending");
  trace.release();
  await pending;
});
