import type { anthropic } from "@ai-sdk/anthropic";
import type { google } from "@ai-sdk/google";
import type { openai } from "@ai-sdk/openai";

export type ProviderModels = {
  anthropic: Parameters<typeof anthropic>[0];
  openai: Parameters<typeof openai>[0];
  google: Parameters<typeof google>[0];
  openrouter: string;
};

export type AskAIProvider = keyof ProviderModels;

type ProviderSelection = {
  [Provider in AskAIProvider]: { provider: Provider; model: ProviderModels[Provider] };
}[AskAIProvider];

export type LimitWindow = "1m" | "1h" | "1d";

export type LimitPreset = {
  max: number;
  per: LimitWindow;
  by?: (request: Request) => string | Promise<string>;
  skipIf?: (request: Request) => boolean | Promise<boolean>;
};

export type LimitCheck = (request: Request) => boolean | Promise<boolean>;

export type AskAIConfig = ProviderSelection & {
  apiKeyEnv?: string;
  baseUrl?: string;
  title?: string;
  placeholder?: string;
  suggestions?: string[];
  instructions?: string;
  hideOn?: string[];
  limit?: LimitPreset | LimitCheck | false;
  maxContextChars?: number;
  maxOutputTokens?: number;
};
