import { syncChangelog, watchChangelog } from "./sync-changelog";

type Mode = "dev" | "start";

const DEFAULT_PORT = 3000;
const MAX_PORT_ATTEMPTS = 50;

function parseMode(value: string | undefined): Mode {
  if (value === "dev" || value === "start") return value;
  throw new Error(`Unknown mode "${value}". Use "dev" or "start".`);
}

function isPortFree(port: number) {
  try {
    const listener = Bun.listen({ hostname: "0.0.0.0", port, socket: { data: () => {} } });
    listener.stop(true);
    return true;
  } catch {
    return false;
  }
}

function findFreePort(preferred: number) {
  for (let port = preferred; port < preferred + MAX_PORT_ATTEMPTS; port++) {
    if (isPortFree(port)) return port;
  }
  throw new Error(
    `No free port found between ${preferred} and ${preferred + MAX_PORT_ATTEMPTS - 1}.`,
  );
}

function openInBrowser(url: string) {
  const command =
    process.platform === "darwin" ? "open" : process.platform === "win32" ? "explorer" : "xdg-open";
  try {
    Bun.spawn([command, url], { stdout: "ignore", stderr: "ignore" }).unref();
    console.log(`  ✓ opened ${url} in the browser`);
  } catch {
    console.error(`  ✗ could not open the browser, visit ${url}`);
  }
}

async function main() {
  const mode = parseMode(process.argv[2]);
  const preferred = Number(process.env.PORT) || DEFAULT_PORT;
  const port = findFreePort(preferred);
  const url = `http://localhost:${port}`;

  if (port !== preferred) {
    console.log(`Port ${preferred} is in use, using ${port} instead.`);
  }

  if (mode === "dev") {
    await syncChangelog();
    watchChangelog();
  }

  const child = Bun.spawn(["bun", "--bun", "next", mode, "-p", String(port)], {
    stdin: "ignore",
    stdout: "inherit",
    stderr: "inherit",
    env: { ...process.env, PORT: String(port) },
  });

  function shutdown() {
    if (process.stdin.isTTY) process.stdin.setRawMode(false);
    child.kill("SIGINT");
  }

  void child.exited.then((code) => process.exit(code));
  process.on("SIGINT", shutdown);
  process.on("SIGTERM", shutdown);

  if (!process.stdin.isTTY) return;

  console.log(`\n  press o to open ${url} in the browser, q to quit\n`);
  process.stdin.setRawMode(true);
  process.stdin.setEncoding("utf8");
  process.stdin.on("data", (key: string) => {
    if (key === "o") openInBrowser(url);
    // \u0003 is ctrl+c, which raw mode delivers as input instead of SIGINT
    if (key === "q" || key === "\u0003") shutdown();
  });
}

try {
  await main();
} catch (error) {
  console.error(error instanceof Error ? error.message : error);
  process.exit(1);
}
