const { describe, it } = require("node:test");
const assert = require("node:assert/strict");
const { buildItem, isSavableUrl } = require("./lib.js");

describe("extension payload (contract with POST /v1/items)", () => {
  it("builds a url item with selection", () => {
    const item = buildItem({ url: "https://example.com/a?x=1", title: " A " }, " quoted ");
    assert.equal(item.source_type, "url");
    assert.equal(item.raw_ref, "https://example.com/a?x=1");
    assert.equal(item.title, "A");
    assert.equal(item.original.selected_text, "quoted");
    assert.equal(item.original.captured_from, "extension");
  });
  it("falls back to url when title is missing", () => {
    const item = buildItem({ url: "https://example.com/a", title: "" }, "");
    assert.equal(item.title, "https://example.com/a");
    assert.ok(!("selected_text" in item.original));
  });
  it("trims long titles and selections to api limits", () => {
    const item = buildItem({ url: "https://e.com", title: "t".repeat(900) }, "s".repeat(9000));
    assert.ok(item.title.length <= 500);
    assert.ok(item.original.selected_text.length <= 5000);
  });
  it("rejects non-web pages", () => {
    assert.equal(isSavableUrl("https://x.com"), true);
    assert.equal(isSavableUrl("chrome://extensions"), false);
    assert.equal(isSavableUrl("about:blank"), false);
    assert.equal(isSavableUrl(""), false);
  });
});
