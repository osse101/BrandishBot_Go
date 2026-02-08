import { useState, useEffect, useCallback, useRef } from 'react';
import { apiFetch, ApiError } from '../api/client';

interface UseApiState<T> {
  data: T | null;
  error: string | null;
  isLoading: boolean;
}

export function useApi<T>(path: string | null, options?: RequestInit) {
  const [state, setState] = useState<UseApiState<T>>({
    data: null,
    error: null,
    isLoading: !!path,
  });

  const optionsRef = useRef(options);
  optionsRef.current = options;

  useEffect(() => {
    if (!path) return;

    let cancelled = false;
    setState(s => ({ ...s, isLoading: true, error: null }));

    apiFetch<T>(path, optionsRef.current)
      .then(data => {
        if (!cancelled) setState({ data, error: null, isLoading: false });
      })
      .catch(err => {
        if (!cancelled) {
          const message = err instanceof ApiError ? err.message : 'Request failed';
          setState({ data: null, error: message, isLoading: false });
        }
      });

    return () => { cancelled = true; };
  }, [path]);

  const refetch = useCallback(() => {
    if (!path) return;
    setState(s => ({ ...s, isLoading: true, error: null }));

    apiFetch<T>(path, optionsRef.current)
      .then(data => setState({ data, error: null, isLoading: false }))
      .catch(err => {
        const message = err instanceof ApiError ? err.message : 'Request failed';
        setState({ data: null, error: message, isLoading: false });
      });
  }, [path]);

  return { ...state, refetch };
}
