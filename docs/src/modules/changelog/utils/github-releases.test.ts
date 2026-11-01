import { expect, test } from "bun:test";
import { releasesFromGithub } from "./github-releases";

const base = { draft: false, prerelease: false, published_at: "2026-10-01T10:00:00Z" };

test("maps tags, dates and bodies", () => {
  const [release] = releasesFromGithub([
    { ...base, tag_name: "v1.2.0", body: "## What's Changed\r\n* Fix by @a in {#1}" },
  ]);
  expect(release).toEqual({
    version: "1.2.0",
    date: "2026-10-01",
    body: "### What's Changed\n* Fix by @a in \\{#1\\}",
    prerelease: false,
  });
});

test("drops drafts and releases without notes", () => {
  const releases = releasesFromGithub([
    { ...base, tag_name: "v1.0.0", body: "Notes", draft: true },
    { ...base, tag_name: "v1.0.1", body: null },
    { ...base, tag_name: "v1.0.2", body: "Notes" },
  ]);
  expect(releases.map((release) => release.version)).toEqual(["1.0.2"]);
});

test("rejects an unexpected payload", () => {
  expect(() => releasesFromGithub({ message: "Not Found" })).toThrow();
});
