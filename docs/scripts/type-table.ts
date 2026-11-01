import { highlightHast } from "fumadocs-core/highlight";
import { transformerTwoslash } from "fumadocs-twoslash";
import { createFileSystemTypesCache } from "fumadocs-twoslash/cache-fs";
import { getTypeInfo } from "../src/modules/docs/server/type-info";
import { describeType } from "../src/modules/docs/utils/format-type";

const themes = { light: "github-light", dark: "github-dark-default" };

const hover = transformerTwoslash({
  explicitTrigger: true,
  typesCache: createFileSystemTypesCache(),
  twoslashOptions: {
    handbookOptions: { noErrors: true },
    compilerOptions: {
      paths: { "@/*": [`${process.cwd()}/src/*`] },
    },
  },
});

async function render(path: string, name: string) {
  const props = getTypeInfo(path, name);

  return Promise.all(
    props.map(async (prop) => {
      const snippet = describeType(prop.name, prop.type, prop.declarations);
      const tree = snippet
        ? await highlightHast(snippet, {
            lang: "ts",
            themes,
            defaultColor: false,
            meta: { __raw: "twoslash" },
            transformers: [hover],
          })
        : null;

      return {
        name: prop.name,
        required: prop.required,
        type: prop.type,
        description: prop.description,
        tree,
      };
    }),
  );
}

const [path, name] = process.argv.slice(2);
if (!path || !name) throw new Error("Usage: bun scripts/type-table.ts <path> <name>");

process.stdout.write(JSON.stringify(await render(path, name)));
process.exit(0);
