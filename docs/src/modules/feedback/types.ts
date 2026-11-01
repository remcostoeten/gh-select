import * as z from "zod";

export const feedbackSchema = z.object({
  url: z.string().startsWith("/"),
  rating: z.enum(["helpful", "not-helpful"]),
});

export type Feedback = z.infer<typeof feedbackSchema>;

export type FeedbackRating = Feedback["rating"];

export type FeedbackCounts = Record<FeedbackRating, number>;

export type FeedbackStore = {
  save: (feedback: Feedback) => Promise<void>;
  counts: (url: string) => Promise<FeedbackCounts>;
};
