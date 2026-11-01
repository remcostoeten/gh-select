import { describe, expect, test } from "bun:test";
import { themeCss } from "./theme-css";

describe("themeCss", () => {
  test("returns nothing for an empty theme", () => {
    expect(themeCss({ accent: null })).toBe("");
  });

  test("scales every radius step from one length", () => {
    const css = themeCss({ accent: null, radius: "1rem" });
    expect(css).toContain("--radius-xl:calc(1rem * 1.0000);");
    expect(css).toContain("--radius-sm:calc(1rem * 0.3333);");
  });

  test("accepts zero for square corners", () => {
    expect(themeCss({ accent: null, radius: "0" })).toContain("--radius-xl:calc(0px * 1.0000);");
  });

  test("rejects values that are not a length", () => {
    expect(() => themeCss({ accent: null, radius: "1rem; color: red" })).toThrow();
  });
});
