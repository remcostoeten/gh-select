import { describe, expect, test } from "bun:test";
import { extractSources } from "./sources";

describe("extractSources", () => {
  test("lists internal pages once, in order", () => {
    const text =
      "See [Theming](/docs/guides/theming) and [Ask AI](/docs/guides/ask-ai#limits), or [theming again](/docs/guides/theming).";
    expect(extractSources(text)).toEqual([
      { url: "/docs/guides/theming", title: "Theming" },
      { url: "/docs/guides/ask-ai", title: "Ask AI" },
    ]);
  });

  test("ignores external links and plain text", () => {
    expect(extractSources("Read [the SDK](https://ai-sdk.dev) first.")).toEqual([]);
    expect(extractSources("No links here.")).toEqual([]);
  });
});
