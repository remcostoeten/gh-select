import { expect, test } from "bun:test";
import { checkLimit } from "./rate-limit";

function request(ip: string, headers?: Record<string, string>) {
  return new Request("http://localhost/api/ask", {
    method: "POST",
    headers: { "x-forwarded-for": ip, ...headers },
  });
}

test("allows up to max requests per key, then blocks with a retry time", async () => {
  const limit = { max: 2, per: "1m" } as const;
  expect((await checkLimit(request("1.1.1.1"), limit)).allowed).toBe(true);
  expect((await checkLimit(request("1.1.1.1"), limit)).allowed).toBe(true);
  const blocked = await checkLimit(request("1.1.1.1"), limit);
  expect(blocked.allowed).toBe(false);
  expect(blocked.retryAfter).toBeGreaterThan(0);
  expect((await checkLimit(request("2.2.2.2"), limit)).allowed).toBe(true);
});

test("counts by a custom key and skips requests that match skipIf", async () => {
  const limit = {
    max: 1,
    per: "1m",
    by: (incoming: Request) => incoming.headers.get("x-user") ?? "anonymous",
    skipIf: (incoming: Request) => incoming.headers.get("x-plan") === "pro",
  } as const;
  const first = request("3.3.3.3", { "x-user": "ada" });
  expect((await checkLimit(first, limit)).allowed).toBe(true);
  expect((await checkLimit(request("4.4.4.4", { "x-user": "ada" }), limit)).allowed).toBe(false);
  const pro = request("5.5.5.5", { "x-user": "ada", "x-plan": "pro" });
  expect((await checkLimit(pro, limit)).allowed).toBe(true);
});

test("a function decides on its own and false turns the limit off", async () => {
  expect((await checkLimit(request("6.6.6.6"), () => false)).allowed).toBe(false);
  expect((await checkLimit(request("6.6.6.6"), false)).allowed).toBe(true);
});
