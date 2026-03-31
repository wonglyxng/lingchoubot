"use client";

import { useEffect, useRef, useCallback, useState } from "react";

const SSE_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

export interface SSEEvent {
  id: string;
  topic: string;
  event_type: string;
  target_id: string;
  project_id?: string;
  data: unknown;
  timestamp: string;
}

interface UseEventStreamOptions {
  /** SSE topics to subscribe: workflow, approval, tool_call, audit. Empty = all. */
  topics?: string[];
  /** Called when an SSE event arrives. */
  onEvent: (event: SSEEvent) => void;
  /** Polling interval in ms when SSE is unavailable (default: 5000). */
  pollInterval?: number;
  /** Polling function — called as fallback when SSE disconnects. */
  onPoll?: () => void;
  /** Whether to enable the stream (default: true). */
  enabled?: boolean;
}

/**
 * React hook for SSE-first, polling-fallback real-time updates.
 *
 * Connects to /api/v1/events/stream via EventSource. If the connection
 * fails 3 times in a row, degrades to periodic polling via onPoll callback.
 * Always auto-reconnects when SSE recovers.
 */
export function useEventStream({
  topics,
  onEvent,
  pollInterval = 5000,
  onPoll,
  enabled = true,
}: UseEventStreamOptions) {
  const [connected, setConnected] = useState(false);
  const [mode, setMode] = useState<"sse" | "poll">("sse");
  const retriesRef = useRef(0);
  const maxRetries = 3;
  const esRef = useRef<EventSource | null>(null);
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const onEventRef = useRef(onEvent);
  const onPollRef = useRef(onPoll);

  // Keep callback refs fresh without re-triggering effect
  useEffect(() => { onEventRef.current = onEvent; }, [onEvent]);
  useEffect(() => { onPollRef.current = onPoll; }, [onPoll]);

  const stopPoll = useCallback(() => {
    if (pollRef.current) {
      clearInterval(pollRef.current);
      pollRef.current = null;
    }
  }, []);

  const startPoll = useCallback(() => {
    stopPoll();
    if (onPollRef.current) {
      setMode("poll");
      pollRef.current = setInterval(() => {
        onPollRef.current?.();
      }, pollInterval);
    }
  }, [pollInterval, stopPoll]);

  useEffect(() => {
    if (!enabled) return;

    let cancelled = false;

    function connect() {
      if (cancelled) return;

      const params = topics?.length ? `?topics=${topics.join(",")}` : "";
      const url = `${SSE_BASE}/api/v1/events/stream${params}`;

      const es = new EventSource(url);
      esRef.current = es;

      es.onopen = () => {
        if (cancelled) return;
        retriesRef.current = 0;
        setConnected(true);
        setMode("sse");
        stopPoll();
      };

      // Listen to named events matching topics
      const eventTopics = topics?.length ? topics : ["workflow", "approval", "tool_call", "audit"];
      for (const topic of eventTopics) {
        es.addEventListener(topic, (e: MessageEvent) => {
          if (cancelled) return;
          try {
            const parsed: SSEEvent = JSON.parse(e.data);
            onEventRef.current(parsed);
          } catch {
            // ignore malformed events
          }
        });
      }

      es.onerror = () => {
        if (cancelled) return;
        es.close();
        esRef.current = null;
        setConnected(false);
        retriesRef.current++;

        if (retriesRef.current >= maxRetries) {
          // Degrade to polling
          startPoll();
          // Still try to reconnect SSE after a longer delay
          setTimeout(connect, 30000);
        } else {
          // Quick retry
          setTimeout(connect, 2000 * retriesRef.current);
        }
      };
    }

    connect();

    return () => {
      cancelled = true;
      esRef.current?.close();
      esRef.current = null;
      stopPoll();
      setConnected(false);
    };
  }, [enabled, topics?.join(","), startPoll, stopPoll]); // eslint-disable-line react-hooks/exhaustive-deps

  return { connected, mode };
}
