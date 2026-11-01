import { describe, expect, test } from "bun:test";
import { formatType, splitUnion } from "./format-type";

describe("splitUnion", () => {
  test("splits top level members", () => {
    expect(splitUnion("string | false")).toEqual(["string", "false"]);
  });

  test("keeps pipes inside generics, strings and functions", () => {
    expect(splitUnion("Array<A | B> | 'a|b'")).toEqual(["Array<A | B>", "'a|b'"]);
    expect(splitUnion("(value: A | B) => void | null")).toEqual(["(value: A | B) => void", "null"]);
  });
});

describe("formatType", () => {
  test("writes unions one member per line", () => {
    expect(formatType("'a' | 'b'", "mode")).toBe("type Mode =\n  | 'a'\n  | 'b'");
  });

  test("skips single word types", () => {
    expect(formatType("string", "title")).toBeNull();
  });

  test("keeps other types on one line", () => {
    expect(formatType("(id: string) => void", "onSelect")).toBe(
      "type OnSelect = (id: string) => void",
    );
  });
});
