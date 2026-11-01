"use client";

import { Analytics } from "@spoar/sdk/next";
import { AnalyticsProvider } from "@spoar/sdk/react";
import { useState } from "react";
import type { ReactNode } from "react";
import { createDocsAnalytics } from "./client";

type Props = {
  children: ReactNode;
};

export function DocsAnalytics({ children }: Props) {
  const [client] = useState(createDocsAnalytics);
  if (!client) return children;
  return (
    <AnalyticsProvider client={client}>
      <Analytics />
      {children}
    </AnalyticsProvider>
  );
}
