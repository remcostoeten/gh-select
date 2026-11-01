import { createAnalytics } from "@spoar/sdk";
import { botSignals, errors, outboundLinks, scrollDepth, speedInsights } from "@spoar/sdk/plugins";

/**
 * @name createDocsAnalytics
 * @description Builds the Spoar client for the docs from `NEXT_PUBLIC_SPOAR_CONFIG`, or returns
 * null when the variable is missing so local runs and forks send nothing. Pageviews come from the
 * Next adapter in the provider, which reports route templates such as `/docs/[[...slug]]`.
 *
 * @example
 * const analytics = createDocsAnalytics();
 * analytics?.track("copy_install", {});
 */
export function createDocsAnalytics() {
  if (!process.env.NEXT_PUBLIC_SPOAR_CONFIG) return null;
  return createAnalytics({
    pageviews: false,
    plugins: [botSignals(), errors(), outboundLinks(), scrollDepth(), speedInsights()],
  });
}
