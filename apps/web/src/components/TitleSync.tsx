"use client";

import { useEffect } from "react";
import { api } from "@/services/api";

function setFavicon(dataURL: string) {
  let link = Array.from(document.querySelectorAll<HTMLLinkElement>("link[rel*='icon']")).find((l) => l.getAttribute("href")?.startsWith("data:"));
  if (!link) {
    link = document.createElement("link");
    link.setAttribute("rel", "icon");
    document.head.appendChild(link);
  }
  link.setAttribute("href", dataURL);
}

export default function TitleSync() {
  useEffect(() => {
    api.siteSettings().then((settings) => {
      if (settings.platform_name) document.title = settings.platform_name;
      if (settings.favicon_data_url) setFavicon(settings.favicon_data_url);
    }).catch(() => undefined);
  }, []);
  return null;
}