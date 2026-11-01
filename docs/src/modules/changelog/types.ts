type ChangelogSource = { file: string; url?: never } | { url: string; file?: never };

export type ChangelogConfig = ChangelogSource & {
  page: string;
  title: string;
  description: string;
  skipPrereleases?: boolean;
  editUrl?: string | false;
};
