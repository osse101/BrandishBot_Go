import { useState, useCallback } from 'react';
import { getApiKey, setApiKey, clearApiKey, apiGet } from '../api/client';
import type { CacheStats } from '../api/types';

export function useAuth() {
  const [isAuthenticated, setIsAuthenticated] = useState(!!getApiKey());
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const login = useCallback(async (key: string) => {
    setIsLoading(true);
    setError(null);
    // Temporarily set the key so the fetch uses it
    setApiKey(key);
    try {
      await apiGet<CacheStats>('/api/v1/admin/cache/stats');
      setIsAuthenticated(true);
    } catch {
      clearApiKey();
      setIsAuthenticated(false);
      setError('Invalid API key');
    } finally {
      setIsLoading(false);
    }
  }, []);

  const logout = useCallback(() => {
    clearApiKey();
    setIsAuthenticated(false);
    setError(null);
  }, []);

  return { isAuthenticated, isLoading, error, login, logout };
}
