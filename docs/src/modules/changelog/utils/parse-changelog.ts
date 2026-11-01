export type Release = {
  version: string;
  date?: string;
  body: string;
  prerelease?: boolean;
};

const releaseHeading = /^##\s+(.+?)\s*#*\s*$/;
const fence = /^\s*(```|~~~)/;
const isoDate = /\d{4}-\d{2}-\d{2}/;
// Matches a version at the start of a heading: 1.2.0, v1.2.0-beta.1 or Unreleased
const versionStart = /^v?(\d+\.\d+(?:\.\d+)?[\w.+-]*|unreleased)/i;
// Matches a link reference definition for a version, such as [1.2.0]: https://...
const versionDefinition = /^\[v?(?:\d[^\]]*|unreleased)\]:\s+\S+\s*$/i;
// Matches a code span with any number of backticks
const codeSpan = /(`+)(?!`).+?(?<!`)\1(?!`)/g;

function stripLinks(text: string) {
  return text.replace(/\[([^\]]+)\]\([^)]*\)/g, "$1").replace(/\[([^\]]+)\]/g, "$1");
}

function parseHeading(heading: string) {
  const text = stripLinks(heading).replace(/[()]/g, " ");
  const date = isoDate.exec(text)?.[0];
  const rest = text
    .replace(isoDate, "")
    .replace(/\s+-\s+|\s+/g, " ")
    .trim();
  const version = versionStart.exec(rest);
  if (!version?.[1]) return { version: rest.replace(/^[-\s]+|[-\s]+$/g, ""), date };
  const raw = version[1];
  return { version: raw.toLowerCase() === "unreleased" ? "Unreleased" : raw, date };
}

function escapePlain(text: string) {
  return text.replace(/([{}<])/g, "\\$1");
}

function escapeLine(line: string) {
  const withLinks = line.replace(/<(https?:\/\/[^>\s]+)>/g, "[$1]($1)");
  let result = "";
  let last = 0;
  for (const match of withLinks.matchAll(codeSpan)) {
    result += escapePlain(withLinks.slice(last, match.index)) + match[0];
    last = match.index + match[0].length;
  }
  return result + escapePlain(withLinks.slice(last));
}

/**
 * @name toMdxSafe
 * @description Makes a release body safe to embed in MDX by escaping braces and
 * angle brackets outside code, turning autolinks into links and removing HTML
 * comments and version link definitions.
 *
 * @example
 * const body = toMdxSafe(release.body)
 */
export function toMdxSafe(body: string) {
  const lines: string[] = [];
  let inFence = false;
  for (const line of body
    .replace(/\r\n/g, "\n")
    .replace(/<!--[\s\S]*?-->/g, "")
    .split("\n")) {
    if (fence.test(line)) {
      inFence = !inFence;
      lines.push(line);
      continue;
    }
    if (inFence) lines.push(line);
    else if (!versionDefinition.test(line.trim())) lines.push(escapeLine(line));
  }
  return lines.join("\n").trim();
}

/**
 * @name parseChangelog
 * @description Splits a Keep a Changelog or conventional changelog file into
 * releases, one per level 2 heading. Reads the version and ISO date from the
 * heading and makes the body safe to embed in MDX. Releases without a body are
 * dropped.
 *
 * @example
 * const [latest] = parseChangelog(await Bun.file("CHANGELOG.md").text())
 */
export function parseChangelog(markdown: string): Release[] {
  const releases: Release[] = [];
  let current: { heading: string; lines: string[] } | undefined;
  let inFence = false;

  function finish() {
    if (!current) return;
    const body = toMdxSafe(current.lines.join("\n"));
    const { version, date } = parseHeading(current.heading);
    if (body && version) releases.push({ version, date, body });
  }

  for (const line of markdown.replace(/\r\n/g, "\n").split("\n")) {
    if (fence.test(line)) inFence = !inFence;
    const heading = inFence ? null : releaseHeading.exec(line);
    if (heading?.[1]) {
      finish();
      current = { heading: heading[1], lines: [] };
    } else {
      current?.lines.push(line);
    }
  }
  finish();
  return releases;
}

/**
 * @name dropPrereleases
 * @description Removes releases whose version has a semver prerelease suffix,
 * such as `2.0.0-next.1`. Unreleased is kept.
 *
 * @example
 * const stable = dropPrereleases(parseChangelog(markdown))
 */
export function dropPrereleases(releases: Release[]) {
  return releases.filter(
    (release) => !release.prerelease && !/^v?\d+\.\d+(?:\.\d+)?-/.test(release.version),
  );
}

/**
 * @name demoteHeadings
 * @description Turns level 1 and 2 headings outside code fences into level 3,
 * so a release body from an external source cannot add releases of its own to
 * the table of contents.
 *
 * @example
 * const body = demoteHeadings("## What's Changed\n- A")
 */
export function demoteHeadings(body: string) {
  let inFence = false;
  return body
    .replace(/\r\n/g, "\n")
    .split("\n")
    .map((line) => {
      if (fence.test(line)) inFence = !inFence;
      return inFence ? line : line.replace(/^#{1,2}(?=\s)/, "###");
    })
    .join("\n");
}
