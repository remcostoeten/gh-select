import { DocsLayout } from "fumadocs-ui/layouts/docs";
import { Suspense, type ReactNode } from "react";
import { source } from "@/modules/docs/server/source";
import { DocsSidebar, DocsSidebarFallback, Topbar } from "@/modules/navigation";
import { getRepository } from "@/shared/config/repository";

export default function Layout({ children }: { children: ReactNode }) {
  const tree = source.getPageTree();
  const repositoryUrl = getRepository()?.url;

  return (
    <>
      <Topbar />
      <Suspense fallback={<DocsSidebarFallback />}>
        <DocsSidebar tree={tree} repositoryUrl={repositoryUrl} />
      </Suspense>
      <DocsLayout
        tree={tree}
        nav={{ enabled: false }}
        searchToggle={{ enabled: false }}
        themeSwitch={{ enabled: false }}
        sidebar={{ enabled: false }}
        containerProps={{ className: "docs-layout" }}
      >
        {children}
      </DocsLayout>
    </>
  );
}
