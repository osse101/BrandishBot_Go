import { useState, useEffect, useRef, useCallback } from 'react';
import { getApiKey } from '../api/client';
import type { SSEEvent } from '../api/types';

export type ConnectionStatus = 'connecting' | 'connected' | 'disconnected';

const MAX_EVENTS = 500;
const MAX_RECONNECT_DELAY = 30000;

export function useSSE(url: string, enabled: boolean) {
  const [events, setEvents] = useState<SSEEvent[]>([]);
  const [status, setStatus] = useState<ConnectionStatus>('disconnected');
  const abortRef = useRef<AbortController>(undefined);
  const retriesRef = useRef(0);

  const clearEvents = useCallback(() => setEvents([]), []);

  useEffect(() => {
    if (!enabled) {
      setStatus('disconnected');
      return;
    }

    let cancelled = false;

    async function connect() {
      const apiKey = getApiKey();
      if (!apiKey) {
        setStatus('disconnected');
        return;
      }

      abortRef.current?.abort();
      const controller = new AbortController();
      abortRef.current = controller;

      setStatus('connecting');

      try {
        const res = await fetch(url, {
          headers: { 'X-API-Key': apiKey },
          signal: controller.signal,
        });

        if (!res.ok || !res.body) {
          throw new Error(`SSE connection failed: ${res.status}`);
        }

        setStatus('connected');
        retriesRef.current = 0;

        const reader = res.body.getReader();
        const decoder = new TextDecoder();
        let buffer = '';

        while (true) {
          const { done, value } = await reader.read();
          if (done || cancelled) break;

          buffer += decoder.decode(value, { stream: true });
          const lines = buffer.split('\n');
          buffer = lines.pop() ?? '';

          let currentId = '';
          let currentType = '';
          let currentData = '';

          for (const line of lines) {
            if (line.startsWith('id: ')) {
              currentId = line.slice(4).trim();
            } else if (line.startsWith('event: ')) {
              currentType = line.slice(7).trim();
            } else if (line.startsWith('data: ')) {
              currentData = line.slice(6).trim();
            } else if (line === '') {
              // Empty line = end of event
              if (currentData && currentType && currentType !== 'keepalive') {
                try {
                  const parsed = JSON.parse(currentData) as SSEEvent;
                  const event: SSEEvent = {
                    id: currentId || parsed.id,
                    type: currentType || parsed.type,
                    timestamp: parsed.timestamp || Date.now() / 1000,
                    payload: parsed.payload,
                  };
                  setEvents(prev => {
                    const next = [...prev, event];
                    return next.length > MAX_EVENTS ? next.slice(-MAX_EVENTS) : next;
                  });
                } catch {
                  // ignore malformed events
                }
              }
              currentId = '';
              currentType = '';
              currentData = '';
            }
          }
        }
      } catch (err) {
        if (cancelled || (err instanceof DOMException && err.name === 'AbortError')) return;
      }

      if (!cancelled) {
        setStatus('disconnected');
        // Exponential backoff reconnect
        const delay = Math.min(1000 * 2 ** retriesRef.current, MAX_RECONNECT_DELAY);
        retriesRef.current++;
        setTimeout(() => {
          if (!cancelled) connect();
        }, delay);
      }
    }

    connect();

    return () => {
      cancelled = true;
      abortRef.current?.abort();
    };
  }, [url, enabled]);

  return { events, status, clearEvents };
}
