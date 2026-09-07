import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: "standalone",
  poweredByHeader: false,
  turbopack: { root: process.cwd() },
  async rewrites() {
    const backend = process.env.API_PROXY_TARGET ?? "http://backend:8080";
    return [{ source: "/api/:path*", destination: `${backend}/api/:path*` }];
  },
  async headers() {
    return [{ source: "/(.*)", headers: [{ key: "X-Frame-Options", value: "DENY" }, { key: "X-Content-Type-Options", value: "nosniff" }] }];
  },
};

export default nextConfig;
