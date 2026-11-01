import "./globals.css";
import { RootProvider } from "fumadocs-ui/provider/next";
import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import type { ReactNode } from "react";
import { DocsAnalytics } from "@/modules/analytics/provider";
import { DocsSearchDialog } from "@/modules/search/components/search-dialog";
import { searchTags } from "@/modules/search/server/sections";
import { faviconIcons } from "@/shared/config/favicon";
import { site } from "@/shared/config/site";
import { themeCss } from "@/shared/config/theme-css";
import { IconMotion } from "@/shared/icons/icon-motion";

const sans = Geist({ subsets: ["latin"], variable: "--font-sans" });
const mono = Geist_Mono({ subsets: ["latin"], variable: "--font-mono" });

export const metadata: Metadata = {
  metadataBase: new URL(site.url),
  title: { template: `%s | ${site.name}`, default: site.name },
  description: site.description,
  applicationName: site.name,
  keywords: site.keywords,
  icons: faviconIcons(site.favicon),
  authors: [site.author],
  creator: site.author.name,
  openGraph: {
    type: "website",
    siteName: site.name,
    title: site.name,
    description: site.description,
    url: "/",
  },
  twitter: { card: "summary", title: site.name, description: site.description },
};

export default function RootLayout({ children }: { children: ReactNode }) {
  return (
    <html
      lang="en"
      className={`${sans.variable} ${mono.variable} antialiased`}
      suppressHydrationWarning
    >
      <head>
        <style dangerouslySetInnerHTML={{ __html: themeCss(site.theme) }} />
      </head>
      <body className="font-sans">
        <RootProvider search={{ SearchDialog: DocsSearchDialog, options: { tags: searchTags() } }}>
          <DocsAnalytics>{children}</DocsAnalytics>
        </RootProvider>
        <IconMotion />
      </body>
    </html>
  );
}
