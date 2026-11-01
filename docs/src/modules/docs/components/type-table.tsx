import { highlightHast } from "fumadocs-core/highlight";
import { Popup, PopupContent, PopupTrigger } from "fumadocs-twoslash/ui";
import { TypeTable as BaseTypeTable } from "fumadocs-ui/components/type-table";
import type { ComponentProps, ReactNode } from "react";
import { toJsxRuntime } from "hast-util-to-jsx-runtime";
import type { Root } from "hast";
import * as JsxRuntime from "react/jsx-runtime";
import { getTypeTableData } from "../server/type-table-data";
import { describeType } from "../utils/format-type";

type Prop = {
  type: string;
  description?: ReactNode;
  typeDescription?: string;
  typeDescriptionLink?: string;
  default?: string;
  required?: boolean;
  deprecated?: boolean;
  parameters?: { name: string; description: ReactNode }[];
  returns?: ReactNode;
};

type Source = {
  path: string;
  name: string;
  descriptions?: Record<string, string>;
  defaults?: Record<string, string>;
};

type Props = Omit<ComponentProps<typeof BaseTypeTable>, "type"> &
  Partial<Source> & { type?: Record<string, Prop> };

const themes = { light: "github-light", dark: "github-dark-default" };
const blockComponents = { Popup, PopupContent, PopupTrigger };

async function highlightTree(code: string) {
  return highlightHast(code, { lang: "ts", themes, defaultColor: false });
}

async function highlightInline(code: string) {
  return toJsxRuntime(await highlightTree(code), {
    ...JsxRuntime,
    components: { pre: ({ children }) => <span className="prop-code">{children}</span> },
  });
}

function renderTree(tree: Root) {
  return toJsxRuntime(tree, {
    ...JsxRuntime,
    components: {
      ...blockComponents,
      pre: ({ children }) => <span className="prop-code prop-code-block">{children}</span>,
    },
  });
}

async function resolveProp(prop: Prop, block: ReactNode) {
  const [type, defaultValue] = await Promise.all([
    highlightInline(prop.type),
    prop.default ? highlightInline(prop.default) : undefined,
  ]);

  return { ...prop, type, typeDescription: block, default: defaultValue };
}

async function readSource({ path, name, descriptions = {}, defaults = {} }: Source) {
  return getTypeTableData(path, name).map((data) => ({
    name: data.name,
    prop: {
      type: data.type,
      required: data.required,
      description: descriptions[data.name] ?? data.description,
      default: defaults[data.name],
    },
    block: data.tree ? renderTree(data.tree) : undefined,
  }));
}

async function readManual(type: Record<string, Prop>) {
  return Promise.all(
    Object.entries(type).map(async ([name, prop]) => {
      const snippet = prop.typeDescription ?? describeType(name, prop.type, []);
      const block = snippet ? await highlightBlock(snippet) : undefined;
      return { name, prop, block };
    }),
  );
}

async function highlightBlock(code: string) {
  return renderTree(await highlightTree(code));
}

/**
 * @name TypeTable
 * @description Props table whose rows expand to the full type as highlighted
 * TypeScript with hover popups. Pass `type` to write the rows by hand, or
 * `path` and `name` to read them from a type in the source with the compiler,
 * including the declarations of every local type a prop refers to.
 *
 * @example
 * <TypeTable path="src/shared/config/site.ts" name="SiteConfig" />
 */
export async function TypeTable({ type, path, name, descriptions, defaults, ...rest }: Props) {
  const rows =
    path && name
      ? await readSource({ path, name, descriptions, defaults })
      : await readManual(type ?? {});
  const entries = await Promise.all(
    rows.map(async (row) => [row.name, await resolveProp(row.prop, row.block)] as const),
  );

  return <BaseTypeTable type={Object.fromEntries(entries)} {...rest} />;
}
