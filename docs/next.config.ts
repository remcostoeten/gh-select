import { createMDX } from "fumadocs-mdx/next";
import type { NextConfig } from "next";

const config: NextConfig = {
  transpilePackages: ["shiki"],
  async redirects() {
    return [{ source: "/", destination: "/docs", permanent: false }];
  },
  async rewrites() {
    return [
      { source: "/docs.mdx", destination: "/llms.mdx/docs" },
      { source: "/docs/:path*.mdx", destination: "/llms.mdx/docs/:path*" },
    ];
  },
};

export default createMDX()(config);
