import type { MetadataRoute } from "next";

const siteURL = process.env.NEXT_PUBLIC_SITE_URL ?? "https://siap-siap.my.id";

export default function robots(): MetadataRoute.Robots {
  return {
    rules: {
      userAgent: "*",
      allow: "/",
      disallow: ["/dashboard", "/cbt", "/exam", "/api"],
    },
    sitemap: `${siteURL}/sitemap.xml`,
    host: siteURL,
  };
}
