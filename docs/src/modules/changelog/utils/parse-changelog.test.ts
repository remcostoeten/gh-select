import { expect, test } from "bun:test";
import { dropPrereleases, parseChangelog } from "./parse-changelog";

test("reads Keep a Changelog headings", () => {
  const releases = parseChangelog(
    "# Changelog\n\n## [1.2.0] - 2026-10-01\n\n### Added\n\n- One\n\n## [Unreleased]\n\n- Two\n\n[1.2.0]: https://example.com/1.2.0\n[Unreleased]: https://example.com/head\n",
  );
  expect(releases.map(({ version, date }) => ({ version, date }))).toEqual([
    { version: "1.2.0", date: "2026-10-01" },
    { version: "Unreleased", date: undefined },
  ]);
  expect(releases[1]?.body).toBe("- Two");
});

test("reads conventional changelog headings", () => {
  const [release] = parseChangelog(
    "## [2.0.0-beta.1](https://example.com/compare) (2026-10-01)\n\n### Features\n\n* thing\n",
  );
  expect(release).toMatchObject({ version: "2.0.0-beta.1", date: "2026-10-01" });
});

test("reads a v prefix and a trailing flag", () => {
  const [release] = parseChangelog("## v1.0.1 - 2026-01-02 [YANKED]\n\n- Gone\n");
  expect(release).toMatchObject({ version: "1.0.1", date: "2026-01-02" });
});

test("drops releases without a body", () => {
  expect(parseChangelog("## [Unreleased]\n\n## [1.0.0] - 2026-01-01\n\n- First\n")).toHaveLength(1);
});

test("keeps headings inside code fences in the body", () => {
  const [release] = parseChangelog("## 1.0.0\n\n```md\n## Not a release\n```\n");
  expect(release?.body).toContain("## Not a release");
});

test("escapes MDX syntax outside code", () => {
  const [release] = parseChangelog(
    "## 1.0.0\n\n- Fixed `{kept}` and {escaped} in <div> <https://example.com/a>\n<!-- hidden -->\n",
  );
  expect(release?.body).toBe(
    "- Fixed `{kept}` and \\{escaped\\} in \\<div> [https://example.com/a](https://example.com/a)",
  );
});

test("drops prereleases but keeps Unreleased", () => {
  const releases = parseChangelog(
    "## 2.0.0\n\n- A\n\n## 2.0.0-next.1\n\n- B\n\n## Unreleased\n\n- C\n\n## 0.1.0-alpha.2\n\n- D\n",
  );
  expect(dropPrereleases(releases).map((release) => release.version)).toEqual([
    "2.0.0",
    "Unreleased",
  ]);
});
