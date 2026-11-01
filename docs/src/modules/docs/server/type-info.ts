import path from "node:path";
import ts from "typescript";

export type PropInfo = {
  name: string;
  required: boolean;
  type: string;
  declarations: string[];
  description: string;
};

type Project = {
  program: ts.Program;
  checker: ts.TypeChecker;
};

const projects = new Map<string, Project>();

function loadProject(file: string) {
  const cached = projects.get(file);
  if (cached) return cached;

  const configPath = path.join(process.cwd(), "tsconfig.json");
  const config = ts.readConfigFile(configPath, ts.sys.readFile);
  const parsed = ts.parseJsonConfigFileContent(config.config, ts.sys, process.cwd());
  const program = ts.createProgram({ rootNames: [file], options: parsed.options });
  const project = { program, checker: program.getTypeChecker() };

  projects.set(file, project);
  return project;
}

function isDeclaration(node: ts.Node): node is ts.TypeAliasDeclaration | ts.InterfaceDeclaration {
  return ts.isTypeAliasDeclaration(node) || ts.isInterfaceDeclaration(node);
}

function resolveDeclaration(project: Project, name: ts.EntityName) {
  const symbol = project.checker.getSymbolAtLocation(name);
  if (!symbol) return undefined;

  const target =
    symbol.flags & ts.SymbolFlags.Alias ? project.checker.getAliasedSymbol(symbol) : symbol;
  const declaration = target.declarations?.find(isDeclaration);
  if (!declaration) return undefined;

  const file = declaration.getSourceFile();
  if (project.program.isSourceFileFromExternalLibrary(file)) return undefined;
  if (project.program.isSourceFileDefaultLibrary(file)) return undefined;
  return declaration;
}

function stripExport(text: string) {
  return text.replace(/^export\s+/, "");
}

function collectDeclarations(
  project: Project,
  node: ts.Node,
  seen: Set<ts.Node>,
  output: string[],
) {
  ts.forEachChild(node, function visit(child) {
    if (ts.isTypeReferenceNode(child)) {
      const declaration = resolveDeclaration(project, child.typeName);
      if (declaration && !seen.has(declaration)) {
        seen.add(declaration);
        output.push(stripExport(declaration.getText()));
        collectDeclarations(project, declaration, seen, output);
      }
    }
    ts.forEachChild(child, visit);
  });
}

function describeProperty(project: Project, symbol: ts.Symbol): PropInfo {
  const declaration = symbol.declarations?.find(ts.isPropertySignature);
  const typeNode = declaration?.type;
  const declarations: string[] = [];

  if (typeNode) collectDeclarations(project, typeNode, new Set(), declarations);

  const type =
    typeNode?.getText() ??
    project.checker.typeToString(
      project.checker.getTypeOfSymbol(symbol),
      undefined,
      ts.TypeFormatFlags.NoTruncation,
    );

  return {
    name: symbol.getName(),
    required: !(symbol.flags & ts.SymbolFlags.Optional),
    type,
    declarations,
    description: ts.displayPartsToString(symbol.getDocumentationComment(project.checker)),
  };
}

/**
 * @name getTypeInfo
 * @description Reads the properties of an object type from a TypeScript file
 * with the compiler API. Each property comes with its declared type, whether it
 * is required, its JSDoc, and the source text of every local type it refers to.
 *
 * @example
 * const props = getTypeInfo("src/shared/config/site.ts", "SiteConfig");
 */
export function getTypeInfo(file: string, name: string) {
  const absolute = path.resolve(process.cwd(), file);
  const project = loadProject(absolute);
  const source = project.program.getSourceFile(absolute);
  const declaration = source?.statements.find(
    (statement): statement is ts.TypeAliasDeclaration | ts.InterfaceDeclaration =>
      isDeclaration(statement) && statement.name.text === name,
  );
  if (!declaration) throw new Error(`Type "${name}" not found in ${file}`);

  const type = project.checker.getTypeAtLocation(declaration.name);
  return project.checker
    .getPropertiesOfType(type)
    .map((symbol) => describeProperty(project, symbol));
}
