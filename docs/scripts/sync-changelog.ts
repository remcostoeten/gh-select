import { existsSync, watch } from "node:fs";
import { basename, dirname, isAbsolute, join, resolve } from "node:path";
import { Glob } from "bun";
import { resolveChangelogSource } from "../src/modules/changelog/utils/changelog-source";
import { releasesFromGithub } from "../src/modules/changelog/utils/github-releases";
import {
  dropPrereleases,
  parseChangelog,
  type Release,
} from "../src/modules/changelog/utils/parse-changelog";
import {
  generatedMarker,
  renderChangelogPage,
} from "../src/modules/changelog/utils/render-changelog-page";
import { repositoryFileUrl } from "../src/shared/config/repository";
import type { ChangelogConfig } from "../src/modules/changelog/types";
import { site } from "../src/shared/config/site";

const githubPages = 5;

type Result = "written" | "unchanged" | "removed";

type Outcome = {
  page: string;
  result: Result;
};

function outputPath(config: ChangelogConfig) {
  return join(site.contentDir, `${config.page}.mdx`);
}

function defaultEditUrl(config: ChangelogConfig) {
  if (config.editUrl !== undefined) return config.editUrl;
  if (config.file === undefined) return resolveChangelogSource(config.url).editUrl;
  if (isAbsolute(config.file) || config.file.startsWith("..")) return false;
  return repositoryFileUrl(config.file);
}

async function readIfExists(path: string) {
  const file = Bun.file(path);
  return (await file.exists()) ? file.text() : undefined;
}

function githubHeaders() {
  const token = process.env.GITHUB_TOKEN;
  return {
    Accept: "application/vnd.github+json",
    "User-Agent": "docs-boilerplate",
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
  };
}

async function fetchOk(url: string, headers: Record<string, string>) {
  const response = await fetch(url, { headers });
  if (!response.ok) throw new Error(`${url} answered ${response.status}`);
  return response;
}

async function fetchGithubReleases(owner: string, repo: string) {
  const releases: Release[] = [];
  for (let page = 1; page <= githubPages; page++) {
    const url = `https://api.github.com/repos/${owner}/${repo}/releases?per_page=100&page=${page}`;
    const batch = releasesFromGithub(await (await fetchOk(url, githubHeaders())).json());
    releases.push(...batch);
    if (batch.length < 100) break;
  }
  return releases;
}

async function loadReleases(config: ChangelogConfig) {
  if (config.file === undefined) {
    const source = resolveChangelogSource(config.url);
    if (source.kind === "github-releases") return fetchGithubReleases(source.owner, source.repo);
    const markdown = await (await fetchOk(source.url, {})).text();
    return parseChangelog(markdown);
  }
  const markdown = await readIfExists(config.file);
  return markdown === undefined ? undefined : parseChangelog(markdown);
}

async function loadOrKeep(config: ChangelogConfig, existing: string | undefined) {
  if (config.file !== undefined) return loadReleases(config);
  try {
    return await loadReleases(config);
  } catch (error) {
    if (existing === undefined) throw error;
    console.warn(`! ${config.page}: could not fetch ${config.url}, keeping the last page`);
    return undefined;
  }
}

async function syncOne(config: ChangelogConfig): Promise<Outcome | undefined> {
  const output = outputPath(config);
  const existing = await readIfExists(output);
  const generated = existing?.includes(generatedMarker) ?? false;

  if (existing !== undefined && !generated) {
    throw new Error(
      `${output} is written by hand. Remove it from site.changelogs or change its page name to generate ${config.url ?? config.file} somewhere else.`,
    );
  }

  const parsed = await loadOrKeep(config, existing);

  if (parsed === undefined) {
    if (!generated || config.file === undefined) return undefined;
    await Bun.file(output).delete();
    return { page: config.page, result: "removed" };
  }

  const releases = config.skipPrereleases ? dropPrereleases(parsed) : parsed;
  const page = renderChangelogPage(releases, config, defaultEditUrl(config));
  if (page === existing) return { page: config.page, result: "unchanged" };
  await Bun.write(output, page);
  return { page: config.page, result: "written" };
}

async function removeStalePages() {
  const wanted = new Set(site.changelogs.map(outputPath));
  const removed: Outcome[] = [];
  for await (const name of new Glob("*.mdx").scan(site.contentDir)) {
    const path = join(site.contentDir, name);
    if (wanted.has(path)) continue;
    if (!(await Bun.file(path).text()).includes(generatedMarker)) continue;
    await Bun.file(path).delete();
    removed.push({ page: name.replace(/\.mdx$/, ""), result: "removed" });
  }
  return removed;
}

/**
 * @name syncChangelog
 * @description Converts every changelog file listed in `site.changelogs` into a
 * docs page. A missing file is skipped and its stale generated page removed, a
 * generated page that is no longer configured is removed, and a page written by
 * hand is never overwritten.
 *
 * @example
 * const outcomes = await syncChangelog()
 */
export async function syncChangelog() {
  const pages = site.changelogs.map((config) => config.page);
  if (new Set(pages).size !== pages.length) {
    throw new Error("site.changelogs has two entries with the same page name.");
  }
  const outcomes: Outcome[] = [];
  for (const config of site.changelogs) {
    const outcome = await syncOne(config);
    if (outcome) outcomes.push(outcome);
  }
  return [...outcomes, ...(await removeStalePages())];
}

/**
 * @name watchChangelog
 * @description Re-runs `syncChangelog` whenever a changelog file in
 * `site.changelogs` changes, wherever it lives, so edits show up in the dev
 * server.
 *
 * @example
 * watchChangelog()
 */
export function watchChangelog() {
  let timer: ReturnType<typeof setTimeout> | undefined;
  for (const config of site.changelogs) {
    if (!config.file) continue;
    const file = resolve(config.file);
    if (!existsSync(dirname(file))) continue;
    watch(dirname(file), (_event, name) => {
      if (name !== basename(file)) return;
      clearTimeout(timer);
      timer = setTimeout(() => {
        syncChangelog().catch((error: unknown) => console.error(error));
      }, 100);
    });
  }
}

if (import.meta.main) {
  try {
    for (const { page, result } of await syncChangelog()) console.log(`✓ ${page} ${result}`);
  } catch (error) {
    console.error(error instanceof Error ? error.message : error);
    process.exit(1);
  }
}
