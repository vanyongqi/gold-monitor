const test = require("node:test");
const assert = require("node:assert/strict");

const { buildXAxisTicks, formatXAxisLabel, getTradingSessionStatus } = require("../internal/web/static/chart-utils.js");

test("buildXAxisTicks returns distributed labels for history data", () => {
  const data = [
    { time: "2026-03-01", value: 1000 },
    { time: "2026-03-02", value: 1001 },
    { time: "2026-03-03", value: 1002 },
    { time: "2026-03-04", value: 1003 },
    { time: "2026-03-05", value: 1004 },
    { time: "2026-03-06", value: 1005 },
    { time: "2026-03-07", value: 1006 },
  ];

  const ticks = buildXAxisTicks(data, 4, "date");

  assert.deepEqual(
    ticks.map((tick) => tick.label),
    ["03-01", "03-03", "03-05", "03-07"],
  );
});

test("formatXAxisLabel formats realtime timestamps to hour minute", () => {
  assert.equal(formatXAxisLabel("2026-04-02T13:51:21+08:00", "time"), "13:51");
});

test("getTradingSessionStatus returns waiting_night after day session", () => {
  const status = getTradingSessionStatus("2026-04-02T16:50:00+08:00");
  assert.equal(status.key, "waiting_night");
  assert.equal(status.label, "日盘已收盘，等待夜盘");
});

test("getTradingSessionStatus returns night_session at night", () => {
  const status = getTradingSessionStatus("2026-04-02T21:05:00+08:00");
  assert.equal(status.key, "night_session");
});
