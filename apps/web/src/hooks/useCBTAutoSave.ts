"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { APIError, api } from "@/services/api";

export type AutoSaveState = "idle" | "pending" | "saving" | "saved" | "offline" | "error";

export function useCBTAutoSave(userExamID: string | null, debounceMs = 350) {
  const [state, setState] = useState<AutoSaveState>("idle");
  const [pendingCount, setPendingCount] = useState(0);
  const pending = useRef(new Map<string, string>());
  const timers = useRef(new Map<string, ReturnType<typeof setTimeout>>());
  const inFlight = useRef(new Map<string, Promise<boolean>>());
  const mounted = useRef(true);

  const flushOne = useCallback((questionID: string): Promise<boolean> => {
    if (!userExamID) return Promise.resolve(false);
    const existing = inFlight.current.get(questionID);
    if (existing) return existing;
    const request = (async () => {
      while (mounted.current) {
        const selectedOption = pending.current.get(questionID);
        if (!selectedOption) return true;
        pending.current.delete(questionID);
        setPendingCount(pending.current.size);
        setState("saving");
        try {
          await api.syncAnswer(userExamID, questionID, selectedOption);
          setState("saved");
        } catch (error: unknown) {
          if (!pending.current.has(questionID)) pending.current.set(questionID, selectedOption);
          setPendingCount(pending.current.size);
          setState(error instanceof APIError && error.networkError ? "offline" : "error");
          return false;
        }
      }
      return false;
    })().finally(() => {
      inFlight.current.delete(questionID);
    });
    inFlight.current.set(questionID, request);
    return request;
  }, [userExamID]);

  const queueAnswer = useCallback((questionID: string, selectedOption: string) => {
    pending.current.set(questionID, selectedOption);
    setPendingCount(pending.current.size);
    setState(navigator.onLine ? "pending" : "offline");
    const current = timers.current.get(questionID);
    if (current) clearTimeout(current);
    const timer = setTimeout(() => { timers.current.delete(questionID); void flushOne(questionID); }, debounceMs);
    timers.current.set(questionID, timer);
  }, [debounceMs, flushOne]);

  const flushAll = useCallback(async () => {
    timers.current.forEach(clearTimeout);
    timers.current.clear();
    await Promise.all(inFlight.current.values());
    const snapshot = [...pending.current.entries()];
    pending.current.clear();
    setPendingCount(0);
    if (!userExamID || snapshot.length === 0) return;
    setState("saving");
    try {
      await Promise.all(snapshot.map(([questionID, option]) => api.syncAnswer(userExamID, questionID, option)));
      if (mounted.current) setState("saved");
    } catch (error) {
      snapshot.forEach(([questionID, option]) => { if (!pending.current.has(questionID)) pending.current.set(questionID, option); });
      setPendingCount(pending.current.size);
      if (mounted.current) setState(error instanceof APIError && error.networkError ? "offline" : "error");
      throw error;
    }
  }, [userExamID]);

  useEffect(() => {
    const retry = () => { for (const questionID of pending.current.keys()) void flushOne(questionID); };
    window.addEventListener("online", retry);
    return () => window.removeEventListener("online", retry);
  }, [flushOne]);

  useEffect(() => {
    mounted.current = true;
    const activeTimers = timers.current;
    return () => { mounted.current = false; activeTimers.forEach(clearTimeout); };
  }, []);

  return { queueAnswer, flushAll, state, pendingCount };
}
