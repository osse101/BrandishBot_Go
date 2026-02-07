import { useState, useEffect, useRef, useCallback } from 'react';
import { apiFetch, ApiError } from '../api/client';

export function usePolling<T>(path: string | null, intervalMs: number) {
  const [data, setData] = useState<T | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(!!path);
  const intervalRef = useRef<ReturnType<typeof setInterval>>(undefined);

  const doFetch = useCallback(async () => {
    if (!path) return;
    try {
      const result = await apiFetch<T>(path);
      setData(result);
      setError(null);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Request failed');
    } finally {
      setIsLoading(false);
    }
  }, [path]);

  useEffect(() => {
    if (!path) return;
    setIsLoading(true);
    doFetch();

    intervalRef.current = setInterval(doFetch, intervalMs);
    return () => {
      if (intervalRef.current) clearInterval(intervalRef.current);
    };
  }, [path, intervalMs, doFetch]);

  return { data, error, isLoading };
}
