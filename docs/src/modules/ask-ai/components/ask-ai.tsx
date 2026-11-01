"use client";

import { Chat, useChat } from "@ai-sdk/react";
import { DefaultChatTransport, type UIMessage } from "ai";
import { AnimatePresence, motion } from "motion/react";
import { useEffect, useRef, useState } from "react";
import { useStickToBottom } from "use-stick-to-bottom";
import { CloseIcon, SparkleIcon } from "@/shared/icons/icons";
import { Answer } from "./answer";
import { CopyButton } from "./copy-button";
import { Sources } from "./sources";

type Props = {
  title: string;
  placeholder: string;
  suggestions: string[];
};

const ease = [0.23, 1, 0.32, 1] as const;
const actionClass =
  "pressable inline-flex h-7 items-center gap-1.5 rounded-md border border-fd-foreground/10 px-2.5 text-xs text-fd-foreground/60 hover:border-fd-foreground/20 hover:bg-fd-foreground/4 hover:text-fd-foreground focus-visible:bg-fd-foreground/8 focus-visible:text-fd-foreground";

let sharedChat: Chat<UIMessage> | undefined;
let sharedOpen = false;

function getChat() {
  sharedChat ??= new Chat({
    transport: new DefaultChatTransport({
      api: "/api/ask",
      body: () => ({ url: window.location.pathname }),
    }),
  });
  return sharedChat;
}

function readError(message: string) {
  try {
    const parsed: { error?: string } = JSON.parse(message);
    return parsed.error ?? message;
  } catch {
    return message;
  }
}

export function AskAI({ title, placeholder, suggestions }: Props) {
  const [open, setOpenState] = useState(sharedOpen);
  const [input, setInput] = useState("");
  const { messages, sendMessage, status, stop, error, setMessages } = useChat({
    chat: getChat(),
  });
  const trigger = useRef<HTMLButtonElement>(null);
  const field = useRef<HTMLInputElement>(null);
  const { scrollRef, contentRef, isAtBottom, scrollToBottom } = useStickToBottom();
  const busy = status === "submitted" || status === "streaming";

  useEffect(() => {
    if (open) field.current?.focus();
  }, [open]);

  function setOpen(next: boolean) {
    sharedOpen = next;
    setOpenState(next);
  }

  function close() {
    setOpen(false);
    trigger.current?.focus();
  }

  function ask(text: string) {
    const question = text.trim();
    if (!question || busy) return;
    void sendMessage({ text: question });
    setInput("");
  }

  return (
    <>
      <button
        ref={trigger}
        type="button"
        aria-expanded={open}
        onClick={() => setOpen(!open)}
        className={actionClass}
      >
        <SparkleIcon className="size-3.5" />
        Ask AI
      </button>
      <AnimatePresence>
        {open ? (
          <motion.dialog
            open
            aria-label={title}
            onKeyDown={(event) => {
              if (event.key === "Escape") close();
            }}
            initial={{ opacity: 0, transform: "translateY(8px) scale(0.98)" }}
            animate={{
              opacity: 1,
              transform: "translateY(0px) scale(1)",
              transition: { duration: 0.18, ease },
            }}
            exit={{
              opacity: 0,
              transform: "translateY(4px) scale(0.98)",
              transition: { duration: 0.12, ease },
            }}
            className="not-prose fixed inset-auto right-4 bottom-4 z-40 m-0 flex p-0 text-fd-foreground h-[min(34rem,calc(100dvh-6rem))] w-[min(26rem,calc(100vw-2rem))] flex-col overflow-hidden rounded-xl border border-fd-foreground/10 bg-fd-background text-sm shadow-xl"
          >
            <div className="flex items-center justify-between border-b border-fd-foreground/8 px-3 py-2">
              <span className="font-medium">{title}</span>
              <div className="flex items-center gap-1">
                {messages.length > 0 ? (
                  <button
                    type="button"
                    onClick={() => {
                      stop();
                      setMessages([]);
                    }}
                    className="pressable rounded-md px-2 py-1 text-xs text-fd-foreground/50 hover:text-fd-foreground"
                  >
                    Clear
                  </button>
                ) : null}
                <button
                  type="button"
                  aria-label="Close"
                  onClick={close}
                  className="pressable inline-flex size-7 items-center justify-center rounded-md text-fd-foreground/50 hover:text-fd-foreground"
                >
                  <CloseIcon className="size-3.5" />
                </button>
              </div>
            </div>

            <div className="relative min-h-0 flex-1">
              <div ref={scrollRef} className="h-full overflow-y-auto">
                <div ref={contentRef} className="space-y-3 p-3">
                  {messages.length === 0 ? (
                    <div className="space-y-2">
                      <p className="text-fd-foreground/50">
                        Answers come from this documentation and can be wrong.
                      </p>
                      {suggestions.map((suggestion) => (
                        <button
                          key={suggestion}
                          type="button"
                          onClick={() => ask(suggestion)}
                          className="pressable block w-full rounded-md border border-fd-foreground/10 px-2.5 py-1.5 text-start text-fd-foreground/70 hover:bg-fd-foreground/4 hover:text-fd-foreground"
                        >
                          {suggestion}
                        </button>
                      ))}
                    </div>
                  ) : null}
                  {messages.map((message, index) => {
                    const text = message.parts
                      .map((part) => (part.type === "text" ? part.text : ""))
                      .join("");
                    if (message.role === "user") {
                      return (
                        <p
                          key={message.id}
                          className="ms-auto w-fit max-w-[85%] rounded-lg bg-fd-foreground/6 px-2.5 py-1.5"
                        >
                          {text}
                        </p>
                      );
                    }
                    if (!text) return null;
                    const streaming = busy && index === messages.length - 1;
                    return (
                      <div key={message.id} className="leading-relaxed">
                        <Answer text={text} streaming={streaming} />
                        {streaming ? null : (
                          <div className="mt-1 flex items-start justify-between gap-2">
                            <Sources text={text} />
                            <CopyButton text={text} label="Copy answer" />
                          </div>
                        )}
                      </div>
                    );
                  })}
                  {status === "submitted" ? (
                    <p className="text-fd-foreground/40">Thinking…</p>
                  ) : null}
                  {error ? (
                    <p role="alert" className="text-red-600 dark:text-red-400">
                      {readError(error.message)}
                    </p>
                  ) : null}
                </div>
              </div>
              {isAtBottom ? null : (
                <button
                  type="button"
                  aria-label="Scroll to latest"
                  onClick={() => void scrollToBottom()}
                  className="pressable absolute start-1/2 bottom-2 inline-flex h-7 -translate-x-1/2 items-center rounded-full border border-fd-foreground/10 bg-fd-background px-3 text-xs text-fd-foreground/60 shadow-sm hover:text-fd-foreground"
                >
                  Latest
                </button>
              )}
            </div>

            <form
              onSubmit={(event) => {
                event.preventDefault();
                ask(input);
              }}
              className="flex items-center gap-2 border-t border-fd-foreground/8 p-2"
            >
              <input
                ref={field}
                value={input}
                onChange={(event) => setInput(event.target.value)}
                placeholder={placeholder}
                maxLength={2000}
                className="min-w-0 flex-1 rounded-md bg-transparent px-2 py-1.5 outline-none placeholder:text-fd-foreground/35"
              />
              {busy ? (
                <button type="button" onClick={() => void stop()} className={actionClass}>
                  Stop
                </button>
              ) : (
                <button type="submit" disabled={!input.trim()} className={actionClass}>
                  Send
                </button>
              )}
            </form>
          </motion.dialog>
        ) : null}
      </AnimatePresence>
    </>
  );
}
