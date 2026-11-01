import type { ReactNode } from "react";

type Props = {
  date?: string;
  children: ReactNode;
};

const dateFormat = new Intl.DateTimeFormat("en-US", { dateStyle: "long", timeZone: "UTC" });

function formatDate(date: string) {
  const parsed = new Date(`${date}T00:00:00Z`);
  return Number.isNaN(parsed.getTime()) ? date : dateFormat.format(parsed);
}

/**
 * @name Release
 * @description One changelog entry for MDX pages. Put the version as a level 2
 * heading inside it so it shows up in the table of contents and search.
 *
 * @example
 * <Release date="2026-10-18">
 *
 * ## 1.2.0
 *
 * - Added dark mode.
 *
 * </Release>
 */
export function Release({ date, children }: Props) {
  return (
    <section className="mt-10 grid gap-3 border-t border-fd-foreground/8 pt-8 first:mt-0 first:border-t-0 first:pt-0 md:grid-cols-[9rem_minmax(0,1fr)] md:gap-8">
      <div className="text-sm text-fd-foreground/50 md:sticky md:top-[calc(var(--topbar-height)+1.5rem)] md:self-start">
        {date ? <time dateTime={date}>{formatDate(date)}</time> : null}
      </div>
      <div className="min-w-0 [&>h2]:mt-0">{children}</div>
    </section>
  );
}
