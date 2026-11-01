import "server-only";
import type { Feedback, FeedbackCounts } from "../types";
import { feedbackStore } from "./store";

/**
 * @name handleFeedback
 * @description Stores one vote with the store exported from `server/store.ts`
 * and returns the new totals for the page. That is the console store unless a
 * database store is installed.
 *
 * @example
 * const counts = await handleFeedback({ url: "/docs", rating: "helpful" })
 */
export async function handleFeedback(feedback: Feedback): Promise<FeedbackCounts> {
  await feedbackStore.save(feedback);
  return feedbackStore.counts(feedback.url);
}

/**
 * @name readFeedbackCounts
 * @description Returns the vote totals of one page from the active store.
 *
 * @example
 * const { helpful, "not-helpful": notHelpful } = await readFeedbackCounts("/docs")
 */
export function readFeedbackCounts(url: string): Promise<FeedbackCounts> {
  return feedbackStore.counts(url);
}
