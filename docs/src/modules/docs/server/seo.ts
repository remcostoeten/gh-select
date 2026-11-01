import "server-only";
import { site } from "@/shared/config/site";
import { source } from "./source";

type DocsPage = NonNullable<ReturnType<typeof source.getPage>>;

const imageFile = "image.png";

/**
 * @name absoluteUrl
 * @description Resolves a site path against the configured site URL.
 *
 * @example
 * absoluteUrl("/docs") // "https://docs.example.com/docs"
 */
export function absoluteUrl(path: string) {
  return new URL(path, site.url).toString();
}

/**
 * @name pageImage
 * @description The Open Graph image route segments and URL for a docs page.
 *
 * @example
 * const { url } = pageImage(page) // "/og/docs/guides/theming/image.png"
 */
export function pageImage(page: DocsPage) {
  const segments = [...page.slugs, imageFile];
  return { segments, url: `/og/docs/${segments.join("/")}` };
}

/**
 * @name pageJsonLd
 * @description Schema.org TechArticle data for a docs page, safe to inline in
 * a script tag.
 *
 * @example
 * <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: pageJsonLd(page) }} />
 */
export function pageJsonLd(page: DocsPage) {
  const data = {
    "@context": "https://schema.org",
    "@type": "TechArticle",
    headline: page.data.title,
    description: page.data.description ?? site.description,
    url: absoluteUrl(page.url),
    image: absoluteUrl(pageImage(page).url),
    author: { "@type": "Person", name: site.author.name, url: site.author.url },
    publisher: { "@type": "Organization", name: site.name, url: site.url },
  };
  return JSON.stringify(data).replace(/</g, "\\u003c");
}
