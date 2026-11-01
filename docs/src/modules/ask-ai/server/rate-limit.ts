import type { AskAIConfig, LimitPreset, LimitWindow } from "../types";

const windowMs: Record<LimitWindow, number> = { "1m": 60_000, "1h": 3_600_000, "1d": 86_400_000 };

const defaultLimit: LimitPreset = { max: 20, per: "1h" };

const maxTrackedClients = 5000;

type Bucket = { count: number; resetAt: number };

const buckets = new Map<string, Bucket>();

/**
 * @name clientIp
 * @description The caller's IP address from the proxy headers, or "unknown"
 * when there are none, in which case all callers share one bucket.
 *
 * @example
 * const key = clientIp(request)
 */
export function clientIp(request: Request) {
  const forwarded = request.headers.get("x-forwarded-for")?.split(",")[0]?.trim();
  return forwarded || request.headers.get("x-real-ip") || "unknown";
}

function prune(now: number) {
  for (const [key, bucket] of buckets) {
    if (bucket.resetAt <= now) buckets.delete(key);
  }
  if (buckets.size > maxTrackedClients) buckets.clear();
}

async function consume(request: Request, preset: LimitPreset) {
  if (await preset.skipIf?.(request)) return { allowed: true, retryAfter: 0 };
  const now = Date.now();
  const key = await (preset.by ?? clientIp)(request);
  const current = buckets.get(key);
  if (!current || current.resetAt <= now) {
    if (buckets.size >= maxTrackedClients) prune(now);
    buckets.set(key, { count: 1, resetAt: now + windowMs[preset.per] });
    return { allowed: true, retryAfter: 0 };
  }
  current.count += 1;
  return {
    allowed: current.count <= preset.max,
    retryAfter: Math.ceil((current.resetAt - now) / 1000),
  };
}

/**
 * @name checkLimit
 * @description Decides whether a request may use Ask AI. A preset counts
 * requests per key in memory, which is per server instance. A function decides
 * on its own, for example against Redis, and a limit of false allows everything.
 * Without a limit, 20 requests per hour per IP apply.
 *
 * @example
 * const { allowed, retryAfter } = await checkLimit(request, config.limit)
 */
export async function checkLimit(request: Request, limit: AskAIConfig["limit"]) {
  if (limit === false) return { allowed: true, retryAfter: 0 };
  if (typeof limit === "function") return { allowed: await limit(request), retryAfter: 60 };
  return consume(request, limit ?? defaultLimit);
}
