import type { ComponentType, SVGProps } from "react";
import type { AskAIConfig } from "@/modules/ask-ai/types";
import type { ChangelogConfig } from "@/modules/changelog/types";
import type { FooterConfig } from "@/modules/footer/types";

export type LogoSource = string | ComponentType<SVGProps<SVGSVGElement>>;

export type FaviconConfig = string | { light: string; dark: string };

export type LogoConfig = {
  src: LogoSource;
  darkSrc?: LogoSource;
  alt?: string;
  width: number;
  height: number;
  hideName?: boolean;
};

export type AccentConfig = {
  light: string;
  dark: string;
};

export type ThemeConfig = {
  accent: AccentConfig | null;
  radius?: string;
};

export type SiteConfig = {
  name: string;
  description: string;
  url: string;
  repository?: string | false;
  branch?: string;
  contentDir: string;
  author: { name: string; url: string };
  keywords: string[];
  logo: LogoConfig | null;
  favicon?: FaviconConfig;
  theme: ThemeConfig;
  pageFooter: { editLink: boolean; lastUpdated: boolean; feedback: boolean; prevNext: boolean };
  siteFooter: FooterConfig | null;
  changelogs: ChangelogConfig[];
  askAI: AskAIConfig | null;
};

export const site: SiteConfig = {
  name: "gh-select",
  description:
    "Documentation for gh-select, a GitHub CLI extension to find, browse and clone GitHub repositories from one terminal screen.",
  url: process.env.NEXT_PUBLIC_SITE_URL ?? "http://localhost:3000",
  contentDir: "content/docs",
  author: { name: "Remco Stoeten", url: "https://remcostoeten.com" },
  keywords: ["gh-select", "gh extension", "github cli", "tui", "git clone", "sparse checkout"],
  logo: null,
  theme: { accent: { light: "#af3a03", dark: "#fe8019" } },
  pageFooter: {
    editLink: true,
    lastUpdated: true,
    feedback: false,
    prevNext: true,
  },
  siteFooter: {
    brand: {
      description:
        "Documentation for gh-select, a GitHub CLI extension to find, browse and clone GitHub repositories from one terminal screen.",
    },
    columns: [
      {
        title: "Docs",
        links: [
          { label: "Introduction", href: "/docs" },
          { label: "Commands", href: "/docs/reference/commands" },
          { label: "Keybindings", href: "/docs/reference/keybindings" },
          { label: "Changelog", href: "/docs/changelog" },
        ],
      },
      {
        title: "Project",
        links: [
          { label: "Source", href: "https://github.com/remcostoeten/gh-select", icon: "GitHub" },
          {
            label: "Releases",
            href: "https://github.com/remcostoeten/gh-select/releases",
          },
          {
            label: "Report an issue",
            href: "https://github.com/remcostoeten/gh-select/issues/new",
          },
        ],
      },
    ],
    bottom: { copyright: "© {year} Remco Stoeten" },
  },
  askAI: null,
  changelogs: [
    {
      file: "../CHANGELOG.md",
      page: "changelog",
      title: "Changelog",
      description: "What changed in each release.",
    },
  ],
};
