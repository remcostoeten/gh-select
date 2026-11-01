import * as z from "zod";
import { demoteHeadings, type Release, toMdxSafe } from "./parse-changelog";

const githubReleases = z.array(
  z.object({
    tag_name: z.string(),
    body: z.string().nullable(),
    draft: z.boolean(),
    prerelease: z.boolean(),
    published_at: z.string().nullable(),
  }),
);

/**
 * @name releasesFromGithub
 * @description Converts the response of the GitHub releases API into releases.
 * Drafts and releases without notes are dropped, a leading `v` is removed from
 * the tag and release notes get their headings demoted.
 *
 * @example
 * const releases = releasesFromGithub(await response.json())
 */
export function releasesFromGithub(payload: unknown): Release[] {
  const releases: Release[] = [];
  for (const item of githubReleases.parse(payload)) {
    const body = toMdxSafe(demoteHeadings(item.body ?? ""));
    if (item.draft || !body) continue;
    releases.push({
      version: item.tag_name.replace(/^v(?=\d)/, ""),
      date: item.published_at?.slice(0, 10),
      body,
      prerelease: item.prerelease,
    });
  }
  return releases;
}
