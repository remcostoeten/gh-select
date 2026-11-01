"use client";

import { useEffect, useState } from "react";
import { CheckIcon, CopyIcon } from "@/shared/icons/icons";

type Props = { text: string; label: string };

export function CopyButton({ text, label }: Props) {
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    if (!copied) return;
    const timer = setTimeout(() => setCopied(false), 1500);
    return () => clearTimeout(timer);
  }, [copied]);

  return (
    <button
      type="button"
      aria-label={copied ? "Copied" : label}
      onClick={() => {
        void navigator.clipboard.writeText(text).then(() => setCopied(true));
      }}
      className="pressable inline-flex size-6 items-center justify-center rounded-md text-fd-foreground/40 hover:text-fd-foreground focus-visible:text-fd-foreground"
    >
      {copied ? <CheckIcon className="size-3.5" /> : <CopyIcon className="size-3.5" />}
    </button>
  );
}
