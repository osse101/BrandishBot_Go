import { apiPost } from '../../api/client';
import { useToast } from '../shared/Toast';
import { useApi } from '../../hooks/useApi';
import { JsonViewer } from '../shared/JsonViewer';
import { useState } from 'react';
import type { CacheStats } from '../../api/types';

export function CachePanel() {
  const toast = useToast();
  const cacheStats = useApi<CacheStats>('/api/v1/admin/cache/stats');

  return (
    <div className="space-y-6">
      {/* Cache Stats */}
      <div className="bg-gray-900 rounded-lg border border-gray-800 p-4">
        <div className="flex items-center justify-between mb-3">
          <h3 className="text-sm font-medium text-gray-300">Cache Statistics</h3>
          <button
            onClick={() => cacheStats.refetch()}
            className="text-xs text-gray-500 hover:text-gray-300 transition-colors"
          >
            Refresh
          </button>
        </div>
        {cacheStats.isLoading && <p className="text-sm text-gray-500">Loading...</p>}
        {cacheStats.data && <JsonViewer data={cacheStats.data} defaultExpanded />}
        {cacheStats.error && <p className="text-sm text-red-400">{cacheStats.error}</p>}
      </div>

      {/* Actions */}
      <div className="bg-gray-900 rounded-lg border border-gray-800 p-4">
        <h3 className="text-sm font-medium text-gray-300 mb-3">Cache Actions</h3>
        <div className="flex gap-2">
          <CacheActionButton
            label="Reload Aliases"
            onClick={async () => {
              await apiPost('/api/v1/admin/reload-aliases');
              toast.success('Aliases reloaded');
            }}
          />
          <CacheActionButton
            label="Reload Voting Weights"
            onClick={async () => {
              await apiPost('/api/v1/admin/progression/reload-weights');
              toast.success('Voting weights reloaded');
            }}
          />
        </div>
      </div>
    </div>
  );
}

function CacheActionButton({ label, onClick }: { label: string; onClick: () => Promise<void> }) {
  const [loading, setLoading] = useState(false);
  const toast = useToast();

  const handleClick = async () => {
    setLoading(true);
    try {
      await onClick();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Action failed');
    } finally {
      setLoading(false);
    }
  };

  return (
    <button
      onClick={handleClick}
      disabled={loading}
      className="px-3 py-1.5 text-sm rounded-md bg-blue-600 text-white hover:bg-blue-500 disabled:opacity-50 transition-colors"
    >
      {loading ? '...' : label}
    </button>
  );
}
