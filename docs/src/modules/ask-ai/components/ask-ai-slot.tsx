import { isHiddenOn } from "@/shared/helpers/is-hidden-on";
import type { AskAIConfig } from "../types";
import { AskAI } from "./ask-ai";

type Props = {
  config: AskAIConfig | null;
  url: string;
  enabled?: boolean;
};

/**
 * @name AskAISlot
 * @description The Ask AI button for a docs page. Renders nothing unless
 * `config` is set, the page does not set `askAI: false` and its URL does not
 * match `hideOn`.
 *
 * @example
 * <PageActions markdownUrl={markdownUrl}>
 *   <AskAISlot config={site.askAI} url={page.url} enabled={page.data.askAI} />
 * </PageActions>
 */
export function AskAISlot({ config, url, enabled }: Props) {
  if (!config || enabled === false || isHiddenOn(url, config.hideOn)) return null;

  return (
    <AskAI
      title={config.title ?? "Ask AI"}
      placeholder={config.placeholder ?? "Ask about these docs"}
      suggestions={config.suggestions ?? []}
    />
  );
}
