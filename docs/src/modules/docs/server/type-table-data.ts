import "server-only";
import { execFileSync } from "node:child_process";
import { readdirSync, statSync } from "node:fs";
import path from "node:path";
import type { Root } from "hast";

export type PropData = {
  name: string;
  required: boolean;
  type: string;
  description: string;
  tree: Root | null;
};

type CacheEntry = {
  stamp: number;
  props: PropData[];
};

const cache = new Map<string, CacheEntry>();

function newestChange(directory: string): number {
  let newest = 0;
  for (const entry of readdirSync(directory, { withFileTypes: true })) {
    const full = path.join(directory, entry.name);
    const time = entry.isDirectory() ? newestChange(full) : statSync(full).mtimeMs;
    if (time > newest) newest = time;
  }
  return newest;
}

/**
 * @name getTypeTableData
 * @description Reads the props of a type from the source and renders each
 * expanded type as highlighted TypeScript with Twoslash hovers. The work runs
 * in a separate Bun process because the compiler and Twoslash cannot be
 * bundled into the Next.js server. Results are cached until a file in `src`
 * changes.
 *
 * @example
 * const props = getTypeTableData("src/shared/config/site.ts", "SiteConfig");
 */
export function getTypeTableData(file: string, name: string) {
  const key = `${file}#${name}`;
  const stamp = newestChange(path.join(process.cwd(), "src"));
  const cached = cache.get(key);
  if (cached && cached.stamp === stamp) return cached.props;

  const output = execFileSync("bun", ["scripts/type-table.ts", file, name], {
    cwd: process.cwd(),
    encoding: "utf8",
    maxBuffer: 64 * 1024 * 1024,
  });
  const props: PropData[] = JSON.parse(output);

  cache.set(key, { stamp, props });
  return props;
}
