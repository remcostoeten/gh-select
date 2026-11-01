"use server";

import type { ActionResult } from "@/shared/types/result";
import { type Feedback, type FeedbackCounts, feedbackSchema } from "../types";
import { handleFeedback, readFeedbackCounts } from "./feedback-handler";

/**
 * @name submitFeedback
 * @description Validates and stores one vote, returning the new totals for the
 * page in a result the widget can show without throwing.
 *
 * @example
 * const result = await submitFeedback({ url: "/docs", rating: "helpful" })
 */
export async function submitFeedback(input: Feedback): Promise<ActionResult<FeedbackCounts>> {
  const parsed = feedbackSchema.safeParse(input);
  if (!parsed.success) return { ok: false, error: "That feedback could not be read." };

  try {
    return { ok: true, data: await handleFeedback(parsed.data) };
  } catch (error) {
    console.error("[feedback]", error);
    return { ok: false, error: "Feedback could not be saved. Try again later." };
  }
}

/**
 * @name getFeedbackCounts
 * @description Returns the vote totals of a page, or an error result when the
 * store cannot be read.
 *
 * @example
 * const result = await getFeedbackCounts("/docs")
 */
export async function getFeedbackCounts(url: string): Promise<ActionResult<FeedbackCounts>> {
  if (!url.startsWith("/")) return { ok: false, error: "That page could not be read." };

  try {
    return { ok: true, data: await readFeedbackCounts(url) };
  } catch (error) {
    console.error("[feedback]", error);
    return { ok: false, error: "Feedback could not be loaded." };
  }
}
