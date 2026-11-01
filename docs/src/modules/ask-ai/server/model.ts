import type { LanguageModel } from "ai";
import type { AskAIConfig, AskAIProvider } from "../types";

const defaultKeyEnv: Record<AskAIProvider, string> = {
  anthropic: "ANTHROPIC_API_KEY",
  openai: "OPENAI_API_KEY",
  google: "GOOGLE_GENERATIVE_AI_API_KEY",
  openrouter: "OPENROUTER_API_KEY",
};

export class MissingKeyError extends Error {
  constructor(readonly variable: string) {
    super(`Ask AI needs the ${variable} environment variable.`);
  }
}

/**
 * @name loadModel
 * @description Creates the language model named in the Ask AI config. The API
 * key is read from the provider's usual environment variable, or from
 * `apiKeyEnv`, and the provider package is only loaded when it is used.
 *
 * @example
 * const model = await loadModel(site.askAI)
 */
export async function loadModel(config: AskAIConfig): Promise<LanguageModel> {
  const variable = config.apiKeyEnv ?? defaultKeyEnv[config.provider];
  const apiKey = process.env[variable];
  if (!apiKey) throw new MissingKeyError(variable);

  switch (config.provider) {
    case "anthropic": {
      const { createAnthropic } = await import("@ai-sdk/anthropic");
      return createAnthropic({ apiKey, baseURL: config.baseUrl })(config.model);
    }
    case "openai": {
      const { createOpenAI } = await import("@ai-sdk/openai");
      const provider = createOpenAI({ apiKey, baseURL: config.baseUrl });
      return config.baseUrl ? provider.chat(config.model) : provider(config.model);
    }
    case "google": {
      const { createGoogleGenerativeAI } = await import("@ai-sdk/google");
      return createGoogleGenerativeAI({ apiKey, baseURL: config.baseUrl })(config.model);
    }
    case "openrouter": {
      const { createOpenRouter } = await import("@openrouter/ai-sdk-provider");
      return createOpenRouter({ apiKey, baseURL: config.baseUrl })(config.model);
    }
  }
}
