import { notFound } from "next/navigation";
import { ImageResponse } from "next/og";
import { pageImage } from "@/modules/docs/server/seo";
import { source } from "@/modules/docs/server/source";
import { site } from "@/shared/config/site";

export async function GET(_request: Request, { params }: RouteContext<"/og/docs/[...slug]">) {
  const { slug } = await params;
  const page = source.getPage(slug.slice(0, -1));
  if (!page) notFound();

  return new ImageResponse(
    <div
      style={{
        display: "flex",
        flexDirection: "column",
        justifyContent: "space-between",
        width: "100%",
        height: "100%",
        padding: "72px 80px",
        background: "#0a0a0a",
        color: "#fafafa",
        fontFamily: "monospace",
      }}
    >
      <div
        style={{
          display: "flex",
          alignItems: "center",
          gap: 16,
          fontSize: 24,
          letterSpacing: 4,
          textTransform: "uppercase",
          color: "#a3a3a3",
        }}
      >
        <div style={{ width: 12, height: 12, background: "#fafafa" }} />
        {site.name}
      </div>
      <div style={{ display: "flex", flexDirection: "column", gap: 24 }}>
        <div style={{ fontSize: 72, fontWeight: 700, lineHeight: 1.05, letterSpacing: -2 }}>
          {page.data.title}
        </div>
        {page.data.description ? (
          <div style={{ fontSize: 30, lineHeight: 1.4, color: "#a3a3a3", maxWidth: 900 }}>
            {page.data.description}
          </div>
        ) : null}
      </div>
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          borderTop: "1px solid #262626",
          paddingTop: 28,
          fontSize: 22,
          color: "#737373",
        }}
      >
        <span>{new URL(site.url).host}</span>
        <span>{page.url}</span>
      </div>
    </div>,
    { width: 1200, height: 630 },
  );
}

export function generateStaticParams() {
  return source.getPages().map((page) => ({ slug: pageImage(page).segments }));
}
