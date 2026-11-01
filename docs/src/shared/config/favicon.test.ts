import { describe, expect, test } from "bun:test";
import { faviconIcons } from "./favicon";

describe("faviconIcons", () => {
  test("returns nothing without a favicon", () => {
    expect(faviconIcons(undefined)).toBeUndefined();
  });

  test("uses a path as is", () => {
    expect(faviconIcons("/favicon.svg")).toEqual({ icon: "/favicon.svg" });
  });

  test("turns inline SVG into a data URL", () => {
    expect(faviconIcons('<svg viewBox="0 0 1 1"></svg>')).toEqual({
      icon: `data:image/svg+xml,${encodeURIComponent('<svg viewBox="0 0 1 1"></svg>')}`,
    });
  });

  test("sets icons per color scheme", () => {
    expect(faviconIcons({ light: "/a.svg", dark: "/b.svg" })).toEqual({
      icon: [
        { url: "/a.svg", media: "(prefers-color-scheme: light)" },
        { url: "/b.svg", media: "(prefers-color-scheme: dark)" },
      ],
    });
  });
});
