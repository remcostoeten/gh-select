import Link from "next/link";

type Source = { url: string; title: string };

type Props = { text: string };

const docLink = /\[([^\]]+)\]\((\/[^)\s#?]*)[^)]*\)/g;

/**
 * @name extractSources
 * @description The documentation pages an answer links to, in order of first
 * mention, without duplicates. Each page the model sends the reader to counts
 * as a source.
 *
 * @example
 * const sources = extractSources("See [Theming](/docs/guides/theming).")
 */
export function extractSources(text: string): Source[] {
  const seen = new Map<string, Source>();
  for (const match of text.matchAll(docLink)) {
    const [, title, url] = match;
    if (title && url && !seen.has(url)) seen.set(url, { url, title });
  }
  return [...seen.values()];
}

export function Sources({ text }: Props) {
  const sources = extractSources(text);
  if (sources.length === 0) return null;

  return (
    <details className="mt-2 text-xs text-fd-foreground/50">
      <summary className="pressable inline-flex w-fit cursor-pointer list-none items-center gap-1 hover:text-fd-foreground [&::-webkit-details-marker]:hidden">
        Sources ({sources.length})
      </summary>
      <ul className="mt-1.5 space-y-1">
        {sources.map((source) => (
          <li key={source.url}>
            <Link
              href={source.url}
              className="block truncate rounded-md border border-fd-foreground/10 px-2 py-1 text-fd-foreground/70 hover:bg-fd-foreground/4 hover:text-fd-foreground"
            >
              {source.title}
              <span className="ms-1.5 text-fd-foreground/35">{source.url}</span>
            </Link>
          </li>
        ))}
      </ul>
    </details>
  );
}
