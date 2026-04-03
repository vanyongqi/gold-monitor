const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");

test("multi layout does not push the main chart down to grid row 5", () => {
  const css = fs.readFileSync(path.join(__dirname, "styles.css"), "utf8");

  assert.doesNotMatch(
    css,
    /\.compact-grid\.layout-multi \.chart-card-main\s*\{\s*grid-row:\s*5;\s*\}/m,
  );
});

test("compact grid collapses to a single column on small screens", () => {
  const css = fs.readFileSync(path.join(__dirname, "styles.css"), "utf8");

  assert.match(
    css,
    /@media \(max-width: 1080px\)\s*\{[\s\S]*\.compact-grid\s*\{\s*grid-template-columns:\s*1fr;\s*\}/m,
  );
});

test("summary panel stays on the same desktop row as the main chart", () => {
  const css = fs.readFileSync(path.join(__dirname, "styles.css"), "utf8");

  assert.match(
    css,
    /\.compact-grid\.layout-multi #summary-panel\s*\{[\s\S]*grid-row:\s*2\s*\/\s*span\s*3\s*;/m,
  );
});
