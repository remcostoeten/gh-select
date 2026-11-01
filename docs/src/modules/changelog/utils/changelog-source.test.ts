import { expect, test } from "bun:test";
import { resolveChangelogSource } from "./changelog-source";

test("a repository or releases URL reads GitHub Releases", () => {
  for (const url of ["https://github.com/acme/app", "https://github.com/acme/app/releases/"]) {
    expect(resolveChangelogSource(url)).toEqual({
      kind: "github-releases",
      owner: "acme",
      repo: "app",
      editUrl: "https://github.com/acme/app/releases",
    });
  }
});

test("a GitHub blob URL reads the raw file", () => {
  const url = "https://github.com/acme/app/blob/main/packages/sdk/CHANGELOG.md";
  expect(resolveChangelogSource(url)).toEqual({
    kind: "markdown",
    url: "https://raw.githubusercontent.com/acme/app/main/packages/sdk/CHANGELOG.md",
    editUrl: url,
  });
});

test("any other URL is fetched as Markdown with no edit link", () => {
  expect(resolveChangelogSource("https://example.com/CHANGELOG.md")).toEqual({
    kind: "markdown",
    url: "https://example.com/CHANGELOG.md",
    editUrl: false,
  });
});
