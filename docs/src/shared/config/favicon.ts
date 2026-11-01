import type { Metadata } from "next";
import { inlineSvg } from "./inline-svg";
import type { FaviconConfig } from "./site";

function faviconHref(source: string) {
  return inlineSvg.test(source) ? `data:image/svg+xml,${encodeURIComponent(source)}` : source;
}

/**
 * @name faviconIcons
 * @description Turns `site.favicon` into the Next.js `icons` metadata. A value is a path in
 * `public`, a URL or inline SVG markup. An object sets separate icons for light and dark mode.
 * Returns undefined when no favicon is configured, so the `icon` file convention keeps working.
 *
 * @example
 * export const metadata = { icons: faviconIcons(site.favicon) };
 */
export function faviconIcons(favicon: FaviconConfig | undefined): Metadata["icons"] {
  if (!favicon) return undefined;
  if (typeof favicon === "string") return { icon: faviconHref(favicon) };
  return {
    icon: [
      { url: faviconHref(favicon.light), media: "(prefers-color-scheme: light)" },
      { url: faviconHref(favicon.dark), media: "(prefers-color-scheme: dark)" },
    ],
  };
}
