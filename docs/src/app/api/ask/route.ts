import { createAskHandler } from "@/modules/ask-ai/server/handler";
import { source } from "@/modules/docs/server/source";
import { site } from "@/shared/config/site";

export const maxDuration = 30;

export const POST = createAskHandler({
  config: site.askAI,
  siteName: site.name,
  getPages: () => source.getPages(),
});
