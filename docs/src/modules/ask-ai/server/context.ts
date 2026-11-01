import "server-only";

export type AskAIPage = {
  url: string;
  data: {
    title: string;
    description?: string;
    getText: (type: "processed") => Promise<string>;
  };
};

type IndexedPage = {
  url: string;
  title: string;
  description: string;
  text: string;
  haystack: string;
};

const minTermLength = 3;

let index: Promise<IndexedPage[]> | undefined;

function loadIndex(pages: AskAIPage[]) {
  index ??= Promise.all(
    pages.map(async (page) => {
      const text = await page.data.getText("processed");
      const description = page.data.description ?? "";
      return {
        url: page.url,
        title: page.data.title,
        description,
        text,
        haystack: `${page.data.title} ${description} ${text}`.toLowerCase(),
      };
    }),
  );
  return index;
}

function format(page: IndexedPage, limit: number) {
  const body = page.text.length > limit ? `${page.text.slice(0, limit)}\n[truncated]` : page.text;
  return `# ${page.title} (${page.url})\n\n${body}`;
}

function score(page: IndexedPage, terms: string[]) {
  let total = 0;
  const title = page.title.toLowerCase();
  const description = page.description.toLowerCase();
  for (const term of terms) {
    if (title.includes(term)) total += 8;
    if (description.includes(term)) total += 4;
    total += Math.min(page.haystack.split(term).length - 1, 6);
  }
  return total;
}

/**
 * @name buildContext
 * @description The documentation text handed to the model with a question. A
 * site that fits in `maxChars` is sent whole. A larger one sends the page the
 * reader is on and the pages that best match the question, in order of match.
 *
 * @example
 * const context = await buildContext(source.getPages(), "How do I deploy?", "/docs/guides", 60000)
 */
export async function buildContext(
  source: AskAIPage[],
  question: string,
  currentUrl: string | undefined,
  maxChars: number,
) {
  const pages = await loadIndex(source);
  const total = pages.reduce((sum, page) => sum + page.text.length, 0);
  if (total <= maxChars) return pages.map((page) => format(page, maxChars)).join("\n\n---\n\n");

  const terms = question
    .toLowerCase()
    .split(/[^\p{L}\p{N}]+/u)
    .filter((term) => term.length >= minTermLength);
  const ranked = pages
    .map((page) => ({ page, score: score(page, terms) + (page.url === currentUrl ? 100 : 0) }))
    .filter((entry) => entry.score > 0)
    .sort((a, b) => b.score - a.score);

  const chosen: string[] = [];
  let remaining = maxChars;
  for (const { page } of ranked) {
    if (remaining <= 0) break;
    const share = Math.min(remaining, Math.max(Math.floor(maxChars / 3), 4000));
    const block = format(page, share);
    chosen.push(block);
    remaining -= block.length;
  }
  return chosen.join("\n\n---\n\n");
}
