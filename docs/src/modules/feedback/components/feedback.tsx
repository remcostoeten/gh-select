"use client";

import { ThumbsDownIcon, ThumbsUpIcon } from "lucide-react";
import { useEffect, useState, useTransition } from "react";
import { cn } from "@/shared/helpers/cn";
import { getFeedbackCounts, submitFeedback } from "../server/submit-feedback";
import type { FeedbackCounts, FeedbackRating } from "../types";

type Props = {
  url: string;
};

const choiceClass =
  "pressable inline-flex h-7 items-center justify-center gap-1.5 rounded-md border border-fd-foreground/10 px-2.5 text-xs tabular-nums text-fd-foreground/60 hover:border-fd-foreground/20 hover:bg-fd-foreground/4 hover:text-fd-foreground focus-visible:bg-fd-foreground/8 focus-visible:text-fd-foreground disabled:opacity-50 aria-pressed:border-fd-foreground/30 aria-pressed:bg-fd-foreground/8 aria-pressed:text-fd-foreground";

const options = [
  { rating: "helpful", label: "Helpful", Icon: ThumbsUpIcon },
  { rating: "not-helpful", label: "Not helpful", Icon: ThumbsDownIcon },
] as const;

function votedKey(url: string) {
  return `feedback:${url}`;
}

export function Feedback({ url }: Props) {
  const [counts, setCounts] = useState<FeedbackCounts>();
  const [voted, setVoted] = useState<FeedbackRating>();
  const [error, setError] = useState<string>();
  const [pending, startTransition] = useTransition();

  useEffect(() => {
    const stored = localStorage.getItem(votedKey(url));
    setVoted(stored === "helpful" || stored === "not-helpful" ? stored : undefined);
    setCounts(undefined);
    getFeedbackCounts(url).then((result) => {
      if (result.ok) setCounts(result.data);
    });
  }, [url]);

  function vote(rating: FeedbackRating) {
    setError(undefined);
    startTransition(async () => {
      const result = await submitFeedback({ url, rating });
      if (!result.ok) {
        setError(result.error);
        return;
      }
      localStorage.setItem(votedKey(url), rating);
      setVoted(rating);
      setCounts(result.data);
    });
  }

  return (
    <section
      aria-label="Page feedback"
      className="not-prose mt-12 flex flex-col gap-3 rounded-xl border border-fd-foreground/8 p-4"
    >
      <div className="flex flex-wrap items-center gap-3">
        <p className="mr-auto text-sm text-fd-foreground/70">Was this page helpful?</p>
        {options.map(({ rating, label, Icon }) => (
          <button
            key={rating}
            type="button"
            aria-label={label}
            aria-pressed={voted === rating}
            disabled={pending || voted !== undefined}
            onClick={() => vote(rating)}
            className={cn(choiceClass, counts === undefined && "w-9 px-0")}
          >
            <Icon aria-hidden className="size-3.5" />
            {counts ? counts[rating] : null}
          </button>
        ))}
      </div>
      {error ? (
        <p className="text-xs text-fd-foreground/70" role="alert">
          {error}
        </p>
      ) : null}
    </section>
  );
}
