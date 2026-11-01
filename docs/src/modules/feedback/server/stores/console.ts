import "server-only";
import type { FeedbackCounts, FeedbackStore } from "../../types";

const totals = new Map<string, FeedbackCounts>();

/**
 * @name consoleStore
 * @description Logs each vote to the server console as one JSON line and keeps
 * the totals in memory, so they reset when the server restarts. The default
 * store, it needs no database.
 *
 * @example
 * await consoleStore.save({ url: "/docs", rating: "helpful" })
 */
export const consoleStore: FeedbackStore = {
  async save(feedback) {
    const current = totals.get(feedback.url) ?? { helpful: 0, "not-helpful": 0 };
    totals.set(feedback.url, { ...current, [feedback.rating]: current[feedback.rating] + 1 });
    console.info("[feedback]", JSON.stringify({ ...feedback, at: new Date().toISOString() }));
  },
  async counts(url) {
    return totals.get(url) ?? { helpful: 0, "not-helpful": 0 };
  },
};
