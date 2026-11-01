const openers = "<([{";
const closers = ">)]}";
const quotes = `"'\``;

/**
 * @name splitUnion
 * @description Splits a TypeScript type string on its top level `|` operators.
 * Pipes inside generics, parentheses, brackets, braces and string literals are
 * kept in place.
 *
 * @example
 * splitUnion("string | false") // ["string", "false"]
 * splitUnion("Array<A | B>") // ["Array<A | B>"]
 */
export function splitUnion(type: string) {
  const members: string[] = [];
  let depth = 0;
  let quote = "";
  let start = 0;

  for (let index = 0; index < type.length; index++) {
    const char = type[index];
    const previous = type[index - 1];

    if (quote) {
      if (char === quote && previous !== "\\") quote = "";
      continue;
    }
    if (quotes.includes(char)) quote = char;
    else if (char === ">" && previous === "=") continue;
    else if (openers.includes(char)) depth++;
    else if (closers.includes(char)) depth--;
    else if (char === "|" && depth === 0) {
      members.push(type.slice(start, index).trim());
      start = index + 1;
    }
  }

  members.push(type.slice(start).trim());
  return members.filter(Boolean);
}

/**
 * @name formatType
 * @description Turns a short type string into a `type` alias declaration for
 * display. Unions are written one member per line. Returns null for a single
 * word type such as `string`, which a declaration would only repeat.
 *
 * @example
 * formatType("string | false", "editUrl")
 * // "type EditUrl =\n  | string\n  | false"
 */
export function formatType(type: string, name: string) {
  const members = splitUnion(type);
  const alias = `type ${name.charAt(0).toUpperCase()}${name.slice(1)}`;

  if (members.length > 1) {
    return `${alias} =\n${members.map((member) => `  | ${member}`).join("\n")}`;
  }
  if (/^\w+$/.test(type)) return null;
  return `${alias} = ${type}`;
}

/**
 * @name describeType
 * @description Builds the TypeScript snippet shown when a props row is
 * expanded: the prop type as an alias, followed by the source of every local
 * type it refers to. Returns null when the type is a single word with nothing
 * to expand.
 *
 * @example
 * describeType("logo", "LogoConfig | null", ["type LogoConfig = { src: string }"])
 */
export function describeType(name: string, type: string, declarations: string[]) {
  const main = formatType(type, name);
  if (!main && declarations.length === 0) return null;

  const head = main ?? `type ${name.charAt(0).toUpperCase()}${name.slice(1)} = ${type}`;
  return [head, ...declarations].join("\n\n");
}
