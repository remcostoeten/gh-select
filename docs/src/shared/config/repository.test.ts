import { expect, test } from "bun:test";
import { parseGithubRemote } from "./repository";
import { themeCss } from "./theme-css";

test("reads SSH and HTTPS GitHub remotes", () => {
  for (const remote of [
    "git@github.com:acme/app.git",
    "https://github.com/acme/app.git",
    "https://github.com/acme/app",
    "ssh://git@github.com/acme/app.git",
  ]) {
    expect(parseGithubRemote(remote)).toBe("https://github.com/acme/app");
  }
});

test("drops credentials from a remote", () => {
  expect(parseGithubRemote("https://user:secret@github.com/acme/app.git")).toBe(
    "https://github.com/acme/app",
  );
});

test("ignores remotes that are not on GitHub", () => {
  expect(parseGithubRemote("git@gitlab.com:acme/app.git")).toBeUndefined();
});

test("builds accent rules for both color schemes", () => {
  const css = themeCss({
    accent: { light: "oklch(0.55 0.2 260)", dark: "oklch(0.75 0.15 260)" },
  });
  expect(css).toContain(":root:root{--color-fd-primary:oklch(0.55 0.2 260);");
  expect(css).toContain(":root.dark{--color-fd-primary:oklch(0.75 0.15 260);");
});

test("builds nothing without an accent and rejects unsafe colors", () => {
  expect(themeCss({ accent: null })).toBe("");
  expect(() => themeCss({ accent: { light: "red;}</style>", dark: "blue" } })).toThrow();
});
