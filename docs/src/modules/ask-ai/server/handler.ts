import "server-only";
import {
  convertToModelMessages,
  createUIMessageStreamResponse,
  streamText,
  toUIMessageStream,
  type UIMessage,
} from "ai";
import * as z from "zod";
import type { AskAIConfig } from "../types";
import { type AskAIPage, buildContext } from "./context";
import { loadModel, MissingKeyError } from "./model";
import { checkLimit } from "./rate-limit";

const defaultMaxContextChars = 60_000;
const defaultMaxOutputTokens = 1024;
const maxMessages = 12;
const maxMessageChars = 2000;

const requestSchema = z.object({
  url: z.string().max(300).optional(),
  messages: z
    .array(
      z.object({
        id: z.string().max(100),
        role: z.enum(["user", "assistant"]),
        parts: z.array(z.object({ type: z.string(), text: z.string().optional() })),
      }),
    )
    .min(1),
});

function fail(status: number, error: string, headers?: Record<string, string>) {
  return Response.json({ error }, { status, headers });
}

function toMessages(parsed: z.infer<typeof requestSchema>) {
  return parsed.messages.slice(-maxMessages).map((message): UIMessage => {
    const text = message.parts
      .filter((part) => part.type === "text")
      .map((part) => part.text ?? "")
      .join("")
      .slice(0, maxMessageChars);
    return { id: message.id, role: message.role, parts: [{ type: "text", text }] };
  });
}

function lastQuestion(messages: UIMessage[]) {
  const last = messages.findLast((message) => message.role === "user");
  return last?.parts.map((part) => (part.type === "text" ? part.text : "")).join("") ?? "";
}

type AskHandlerOptions = {
  config: AskAIConfig | null;
  siteName: string;
  getPages: () => AskAIPage[];
};

function buildInstructions(siteName: string, context: string, extra: string | undefined) {
  return [
    `You answer questions about the ${siteName} documentation.`,
    "Use only the documentation below. If it does not contain the answer, say so plainly and do not guess.",
    "Be concise. Use Markdown, put code in fenced blocks, and link pages as Markdown links using the paths in the documentation.",
    extra,
    "Documentation:",
    context,
  ]
    .filter(Boolean)
    .join("\n\n");
}

/**
 * @name createAskHandler
 * @description Creates the `POST /api/ask` handler. It checks the rate limit,
 * loads the documentation that matches the question and streams the model's
 * answer as AI SDK UI messages. It answers 404 when `config` is null.
 *
 * @example
 * export const POST = createAskHandler({
 *   config: site.askAI,
 *   siteName: site.name,
 *   getPages: () => source.getPages(),
 * })
 */
export function createAskHandler({ config, siteName, getPages }: AskHandlerOptions) {
  return async function handleAsk(request: Request) {
    if (!config) return fail(404, "Ask AI is not enabled.");

    const limit = await checkLimit(request, config.limit);
    if (!limit.allowed) {
      return fail(429, "Too many questions. Try again later.", {
        "Retry-After": String(limit.retryAfter),
      });
    }

    const parsed = requestSchema.safeParse(await request.json().catch(() => null));
    if (!parsed.success) return fail(400, "Invalid request.");

    const messages = toMessages(parsed.data);
    const question = lastQuestion(messages);
    if (!question.trim()) return fail(400, "Ask a question first.");

    try {
      const model = await loadModel(config);
      const context = await buildContext(
        getPages(),
        question,
        parsed.data.url,
        config.maxContextChars ?? defaultMaxContextChars,
      );
      const result = streamText({
        model,
        instructions: buildInstructions(siteName, context, config.instructions),
        messages: await convertToModelMessages(messages),
        maxOutputTokens: config.maxOutputTokens ?? defaultMaxOutputTokens,
      });
      return createUIMessageStreamResponse({
        stream: toUIMessageStream({ stream: result.stream }),
      });
    } catch (error) {
      if (error instanceof MissingKeyError) return fail(503, error.message);
      console.error(error);
      return fail(500, "Ask AI failed. Try again.");
    }
  };
}
