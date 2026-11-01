import type { ThemeConfig } from "./site";

const unsafe = /[;{}<>\\]/;

function accentRules(color: string) {
  if (unsafe.test(color)) throw new Error(`site.theme.accent has an unsupported color: ${color}`);
  // White text on dark colors and black text on light ones, from the color's lightness
  const foreground = `oklch(from ${color} calc((0.65 - l) * 1000) 0 0)`;
  return `--color-fd-primary:${color};--color-fd-primary-foreground:${foreground};`;
}

const radiusPattern = /^(0|\d*\.?\d+(px|rem|em))$/;

// Tailwind's default radius scale as multiples of its `xl` step, which cards and callouts use
const radiusSteps = {
  xs: 1 / 6,
  sm: 1 / 3,
  md: 1 / 2,
  lg: 2 / 3,
  xl: 1,
  "2xl": 4 / 3,
  "3xl": 2,
  "4xl": 8 / 3,
};

function radiusRules(input: string) {
  if (!radiusPattern.test(input)) {
    throw new Error(`site.theme.radius must be a length such as "0.75rem" or "0", got: ${input}`);
  }
  const radius = input === "0" ? "0px" : input;
  const steps = Object.entries(radiusSteps)
    .map(([step, factor]) => `--radius-${step}:calc(${radius} * ${factor.toFixed(4)});`)
    .join("");
  return `:root:root{${steps}}`;
}

/**
 * @name themeCss
 * @description CSS for `site.theme`. The accent sets the Fumadocs primary color
 * for light and dark mode with a readable foreground chosen automatically. The
 * radius sets the corner size of cards, callouts, code blocks, search and
 * every other rounded element from one length, scaling the whole radius scale.
 * Returns an empty string when neither is set.
 *
 * @example
 * <style dangerouslySetInnerHTML={{ __html: themeCss(site.theme) }} />
 */
export function themeCss({ accent, radius }: ThemeConfig) {
  const colors = accent
    ? `:root:root{${accentRules(accent.light)}}:root.dark{${accentRules(accent.dark)}}`
    : "";
  return `${colors}${radius === undefined ? "" : radiusRules(radius)}`;
}
