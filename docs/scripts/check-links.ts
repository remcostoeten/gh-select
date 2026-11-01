import { Glob } from "bun";
import { join, relative } from "node:path";
import { site } from "../src/shared/config/site";
import { footerInternalLinks, validateFooter } from "../src/modules/footer/utils/validate-footer";

const contentDir = "content/docs";
const baseUrl = "/docs";

type Page = {
  file: string;
  url: string;
  anchors: Set<string>;
  links: string[];
};

type BrokenLink = {
  file: string;
  link: string;
  reason: string;
};

// Matches Markdown link targets and JSX href attributes
const linkPattern = /\]\(([^)\s]+)\)|href=["{]["']?([^"'}\s]+)["']?}?/g;
// Matches ATX headings outside fenced code
const headingPattern = /^#{1,6}\s+(.+?)\s*$/;

function slugify(text: string) {
  return text
    .toLowerCase()
    .replace(/`/g, "")
    .replace(/[^\p{L}\p{N}\s-]/gu, "")
    .trim()
    .replace(/\s+/g, "-");
}

function fileToUrl(file: string) {
  const path = relative(contentDir, file)
    .replace(/\.mdx?$/, "")
    .replace(/(^|\/)index$/, "");
  return path ? `${baseUrl}/${path}` : baseUrl;
}

function readHeadingsAndLinks(source: string) {
  const anchors = new Set<string>();
  const links: string[] = [];
  let inFence = false;
  for (const line of source.split("\n")) {
    if (line.trimStart().startsWith("```")) {
      inFence = !inFence;
      continue;
    }
    if (inFence) continue;
    const heading = headingPattern.exec(line);
    if (heading?.[1]) anchors.add(slugify(heading[1]));
    for (const match of line.matchAll(linkPattern)) {
      const link = match[1] ?? match[2];
      if (link) links.push(link);
    }
  }
  return { anchors, links };
}

async function loadPages() {
  const pages = new Map<string, Page>();
  for await (const path of new Glob("**/*.{md,mdx}").scan(contentDir)) {
    const file = join(contentDir, path);
    const { anchors, links } = readHeadingsAndLinks(await Bun.file(file).text());
    const url = fileToUrl(file);
    pages.set(url, { file, url, anchors, links });
  }
  return pages;
}

function checkLink(page: Page, link: string, pages: Map<string, Page>): string | undefined {
  if (!link.startsWith("/") && !link.startsWith("#")) return undefined;
  const [path = "", anchor] = link.split("#");
  const target = path === "" ? page : pages.get(path.replace(/\/$/, "") || "/");
  if (path !== "" && !path.startsWith(baseUrl)) return undefined;
  if (!target) return "page does not exist";
  if (anchor && !target.anchors.has(anchor)) return `heading #${anchor} does not exist`;
  return undefined;
}

async function main() {
  const pages = await loadPages();
  const broken: BrokenLink[] = [];
  for (const page of pages.values()) {
    for (const link of page.links) {
      const reason = checkLink(page, link, pages);
      if (reason) broken.push({ file: page.file, link, reason });
    }
  }

  if (site.siteFooter) {
    const footerPage: Page = {
      file: "src/shared/config/site.ts",
      url: "",
      anchors: new Set(),
      links: footerInternalLinks(site.siteFooter),
    };
    for (const link of footerPage.links) {
      const reason = checkLink(footerPage, link, pages);
      if (reason) broken.push({ file: footerPage.file, link, reason });
    }
    for (const problem of validateFooter(site.siteFooter)) {
      broken.push({ file: "src/shared/config/site.ts", link: "", reason: problem });
    }
  }

  if (broken.length === 0) {
    console.log(`✓ checked links in ${pages.size} pages, none broken`);
    return;
  }

  for (const { file, link, reason } of broken) {
    console.error(link ? `✗ ${file}: ${link} (${reason})` : `✗ ${file}: ${reason}`);
  }
  console.error(`\n${broken.length} broken link${broken.length === 1 ? "" : "s"}`);
  process.exit(1);
}

await main();
