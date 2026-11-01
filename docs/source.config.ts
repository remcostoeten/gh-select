import { type RehypeCodeOptions, rehypeCodeDefaultOptions } from "fumadocs-core/mdx-plugins";
import { pageSchema } from "fumadocs-core/source/schema";
import { defineConfig, defineDocs } from "fumadocs-mdx/config";
import lastModified from "fumadocs-mdx/plugins/last-modified";
import { transformerTwoslash } from "fumadocs-twoslash";
import { createFileSystemTypesCache } from "fumadocs-twoslash/cache-fs";
import * as z from "zod";

type Transformer = NonNullable<RehypeCodeOptions["transformers"]>[number];

const transformerLanguage: Transformer = {
  name: "language",
  pre(node) {
    node.properties["data-language"] = this.options.lang;
  },
};

const markdownLanguages = new Set(["md", "markdown", "mdx"]);

function showNotationInMarkdown(transformer: Transformer): Transformer {
  return {
    ...transformer,
    code(node) {
      if (markdownLanguages.has(this.options.lang)) return;
      return transformer.code?.call(this, node);
    },
  };
}

const twoslash = transformerTwoslash({
  typesCache: createFileSystemTypesCache(),
  twoslashOptions: {
    compilerOptions: {
      paths: { "@/*": [`${process.cwd()}/src/*`] },
    },
  },
});

export const docs = defineDocs({
  dir: "content/docs",
  docs: {
    schema: pageSchema.extend({
      sidebarTitle: z.string().min(1).optional(),
      sidebarBadge: z.string().min(1).optional(),
      askAI: z.boolean().optional(),
      siteFooter: z.boolean().optional(),
      tocDepth: z.number().int().min(2).max(6).optional(),
      editUrl: z.union([z.string().min(1), z.literal(false)]).optional(),
    }),
    postprocess: {
      includeProcessedMarkdown: true,
    },
  },
});

export default defineConfig({
  plugins: [lastModified()],
  mdxOptions: {
    rehypeCodeOptions: {
      ...rehypeCodeDefaultOptions,
      themes: { light: "github-light", dark: "github-dark-default" },
      transformers: [
        ...(rehypeCodeDefaultOptions.transformers ?? []).map(showNotationInMarkdown),
        transformerLanguage,
        twoslash,
      ],
    },
  },
});
