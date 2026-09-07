"use client";

import { useEffect, useRef } from "react";

export type AntiCheatEvent = { type: string; occurredAt: string };

export function useAntiCheat(enabled: boolean, report: (event: AntiCheatEvent) => void) {
  const reportRef = useRef(report);
  const lastEventRef = useRef<Record<string, number>>({});

  useEffect(() => {
    reportRef.current = report;
  }, [report]);

  useEffect(() => {
    if (!enabled) return;
    const emit = (type: string) => {
      const now = Date.now();
      if (now - (lastEventRef.current[type] ?? 0) < 750) return;
      lastEventRef.current[type] = now;
      reportRef.current({ type, occurredAt: new Date(now).toISOString() });
    };
    const blockContextMenu = (event: MouseEvent) => { event.preventDefault(); emit("context_menu_blocked"); };
    const blockKeys = (event: KeyboardEvent) => {
      const key = event.key.toLowerCase();
      const clipboard = (event.ctrlKey || event.metaKey) && (key === "c" || key === "v");
      const devtools = event.key === "F12" || ((event.ctrlKey || event.metaKey) && event.shiftKey && ["i", "j", "c"].includes(key)) || ((event.ctrlKey || event.metaKey) && key === "u");
      if (clipboard || devtools) { event.preventDefault(); event.stopPropagation(); emit(devtools ? "devtools_shortcut_blocked" : `clipboard_${key}_blocked`); }
    };
    const visibility = () => { if (document.hidden) emit("tab_hidden"); };
    const blur = () => emit("window_blurred");
    document.addEventListener("contextmenu", blockContextMenu);
    document.addEventListener("keydown", blockKeys, true);
    document.addEventListener("visibilitychange", visibility);
    window.addEventListener("blur", blur);
    return () => {
      document.removeEventListener("contextmenu", blockContextMenu);
      document.removeEventListener("keydown", blockKeys, true);
      document.removeEventListener("visibilitychange", visibility);
      window.removeEventListener("blur", blur);
    };
  }, [enabled]);
}
