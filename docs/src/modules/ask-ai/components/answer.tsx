import { DynamicCodeBlock } from "fumadocs-ui/components/dynamic-codeblock";
import Link from "next/link";
import { isValidElement, type ComponentProps } from "react";
import Markdown, { type Components } from "react-markdown";
import remarkGfm from "remark-gfm";

type Props = { text: string; streaming?: boolean };

function readCode(children: ComponentProps<"pre">["children"]) {
  if (!isValidElement<{ className?: string; children?: unknown }>(children)) return null;
  const { className, children: code } = children.props;
  if (typeof code !== "string") return null;
  return {
    lang: /language-([\w-]+)/.exec(className ?? "")?.[1] ?? "text",
    code: code.replace(/\n$/, ""),
  };
}

function PlainCode({ children }: ComponentProps<"pre">) {
  return (
    <pre className="my-2 overflow-x-auto rounded-md bg-fd-foreground/5 p-2.5 text-xs leading-relaxed">
      {children}
    </pre>
  );
}

function HighlightedCode({ children }: ComponentProps<"pre">) {
  const block = readCode(children);
  if (!block) return <PlainCode>{children}</PlainCode>;
  return (
    <DynamicCodeBlock
      lang={block.lang}
      code={block.code}
      codeblock={{ className: "my-2 text-xs" }}
    />
  );
}

const components: Components = {
  a({ href, children }) {
    if (href?.startsWith("/")) {
      return (
        <Link href={href} className="underline underline-offset-2 hover:text-fd-foreground">
          {children}
        </Link>
      );
    }
    return (
      <a
        href={href}
        target="_blank"
        rel="noreferrer noopener"
        className="underline underline-offset-2 hover:text-fd-foreground"
      >
        {children}
      </a>
    );
  },
  code({ className, children }) {
    return className ? (
      <code className={className}>{children}</code>
    ) : (
      <code className="rounded bg-fd-foreground/8 px-1 py-0.5 font-mono text-[0.85em]">
        {children}
      </code>
    );
  },
  p: ({ children }) => <p className="my-2 first:mt-0 last:mb-0">{children}</p>,
  ul: ({ children }) => <ul className="my-2 list-disc ps-5">{children}</ul>,
  ol: ({ children }) => <ol className="my-2 list-decimal ps-5">{children}</ol>,
  li: ({ children }) => <li className="my-0.5">{children}</li>,
  h1: ({ children }) => <p className="my-2 font-semibold">{children}</p>,
  h2: ({ children }) => <p className="my-2 font-semibold">{children}</p>,
  h3: ({ children }) => <p className="my-2 font-semibold">{children}</p>,
};

const streamingComponents: Components = { ...components, pre: PlainCode };
const finalComponents: Components = { ...components, pre: HighlightedCode };

export function Answer({ text, streaming = false }: Props) {
  return (
    <Markdown
      remarkPlugins={[remarkGfm]}
      components={streaming ? streamingComponents : finalComponents}
    >
      {text}
    </Markdown>
  );
}
