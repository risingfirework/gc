import type { MetadataRoute } from "next";

const siteURL = process.env.NEXT_PUBLIC_SITE_URL ?? "https://siap-siap.my.id";

export default function sitemap(): MetadataRoute.Sitemap {
  const lastModified = new Date();
  return [
    { url: `${siteURL}/`, lastModified, changeFrequency: "weekly", priority: 1 },
    { url: `${siteURL}/login`, lastModified, changeFrequency: "monthly", priority: 0.6 },
    { url: `${siteURL}/terms`, lastModified, changeFrequency: "yearly", priority: 0.3 },
    { url: `${siteURL}/privacy`, lastModified, changeFrequency: "yearly", priority: 0.3 },
  ];
}
