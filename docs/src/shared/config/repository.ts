import { execFileSync } from "node:child_process";
import { site } from "./site";

export type Repository = {
  url: string;
  branch: string;
};

const fallbackBranch = "main";

let cached: Repository | null | undefined;

function git(...args: string[]) {
  try {
    return execFileSync("git", args, {
      encoding: "utf8",
      stdio: ["ignore", "pipe", "ignore"],
      timeout: 5000,
    }).trim();
  } catch {
    return undefined;
  }
}

/**
 * @name parseGithubRemote
 * @description Turns an SSH or HTTPS git remote into a plain `https://github.com`
 * repository URL. Credentials in the remote are dropped. Returns undefined for
 * remotes that are not on GitHub.
 *
 * @example
 * parseGithubRemote("git@github.com:acme/app.git") // "https://github.com/acme/app"
 */
export function parseGithubRemote(remote: string) {
  const match = /github\.com[:/]([^/\s:@]+)\/([^/\s]+?)(?:\.git)?\/?$/.exec(remote.trim());
  return match ? `https://github.com/${match[1]}/${match[2]}` : undefined;
}

function detectUrl() {
  const remote = git("remote", "get-url", "origin");
  const fromRemote = remote ? parseGithubRemote(remote) : undefined;
  if (fromRemote) return fromRemote;
  const { GITHUB_REPOSITORY, VERCEL_GIT_PROVIDER, VERCEL_GIT_REPO_OWNER, VERCEL_GIT_REPO_SLUG } =
    process.env;
  if (GITHUB_REPOSITORY) return `https://github.com/${GITHUB_REPOSITORY}`;
  if (VERCEL_GIT_PROVIDER === "github" && VERCEL_GIT_REPO_OWNER && VERCEL_GIT_REPO_SLUG) {
    return `https://github.com/${VERCEL_GIT_REPO_OWNER}/${VERCEL_GIT_REPO_SLUG}`;
  }
  return undefined;
}

function detectBranch() {
  const remoteHead = git("symbolic-ref", "--short", "refs/remotes/origin/HEAD")?.replace(
    /^origin\//,
    "",
  );
  if (remoteHead) return remoteHead;
  const current = git("rev-parse", "--abbrev-ref", "HEAD");
  if (current && current !== "HEAD") return current;
  return process.env.VERCEL_GIT_COMMIT_REF ?? fallbackBranch;
}

function repositoryPrefix() {
  return git("rev-parse", "--show-prefix") ?? "";
}

/**
 * @name getRepository
 * @description The GitHub repository this site's content lives in, read from
 * the `origin` git remote with the Vercel and GitHub Actions environment as
 * fallback, so no keys or settings are needed. `site.repository` and
 * `site.branch` override it, and `site.repository: false` turns it off.
 * Returns null when there is no repository.
 *
 * @example
 * const repository = getRepository()
 * if (repository) console.log(`${repository.url}/blob/${repository.branch}/README.md`)
 */
export function getRepository() {
  if (cached !== undefined) return cached;
  if (site.repository === false) {
    cached = null;
    return cached;
  }
  const url = site.repository ?? detectUrl();
  cached = url ? { url, branch: site.branch ?? detectBranch() } : null;
  return cached;
}

/**
 * @name repositoryFileUrl
 * @description The GitHub URL of a file in the repository, or false when the
 * site has no repository.
 *
 * @example
 * repositoryFileUrl("content/docs/index.mdx") // "https://github.com/acme/app/blob/main/content/docs/index.mdx"
 */
export function repositoryFileUrl(path: string) {
  const repository = getRepository();
  return repository
    ? `${repository.url}/blob/${repository.branch}/${repositoryPrefix()}${path}`
    : false;
}
