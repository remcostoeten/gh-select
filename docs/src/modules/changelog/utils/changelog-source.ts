export type ChangelogSource =
  | { kind: "github-releases"; owner: string; repo: string; editUrl: string }
  | { kind: "markdown"; url: string; editUrl: string | false };

/**
 * @name resolveChangelogSource
 * @description Works out where a changelog URL points. A GitHub repository or
 * its releases page reads GitHub Releases, a GitHub blob URL reads the raw file,
 * and any other URL is fetched as Markdown.
 *
 * @example
 * resolveChangelogSource("https://github.com/acme/app")
 * // { kind: "github-releases", owner: "acme", repo: "app", ... }
 */
export function resolveChangelogSource(input: string): ChangelogSource {
  const url = new URL(input);
  const [owner, repo, section, ref, ...path] = url.pathname.split("/").filter(Boolean);

  if (url.hostname !== "github.com" || !owner || !repo) {
    return { kind: "markdown", url: url.href, editUrl: false };
  }

  if (section === "blob" && ref && path.length > 0) {
    return {
      kind: "markdown",
      url: `https://raw.githubusercontent.com/${owner}/${repo}/${ref}/${path.join("/")}`,
      editUrl: url.href,
    };
  }

  return {
    kind: "github-releases",
    owner,
    repo,
    editUrl: `https://github.com/${owner}/${repo}/releases`,
  };
}
