const test = require("node:test");
const assert = require("node:assert/strict");

const { buildChartRenderSignature } = require("./chart-utils.js");

test("buildChartRenderSignature returns the same signature for identical chart input", () => {
  const chart = {
    mode: "time",
    valueLabel: "价格",
    width: 900,
    height: 260,
    data: [
      { time: "2026-04-03T09:00:00+08:00", value: 812.11 },
      { time: "2026-04-03T09:01:00+08:00", value: 812.35 },
    ],
    dashedLines: [{ value: 800, color: "#b45309", label: "回本线" }],
  };

  const first = buildChartRenderSignature(chart);
  const second = buildChartRenderSignature({ ...chart });

  assert.equal(first, second);
});

test("buildChartRenderSignature changes when chart data changes", () => {
  const base = {
    mode: "date",
    valueLabel: "收益",
    width: 900,
    height: 320,
    data: [
      { time: "2026-04-01", value: 10 },
      { time: "2026-04-02", value: 12 },
    ],
    dashedLines: [{ value: 0, color: "#b45309", label: "收益为 0" }],
  };

  const previous = buildChartRenderSignature(base);
  const next = buildChartRenderSignature({
    ...base,
    data: [
      { time: "2026-04-01", value: 10 },
      { time: "2026-04-02", value: 13 },
    ],
  });

  assert.notEqual(previous, next);
});
