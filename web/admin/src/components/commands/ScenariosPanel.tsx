import { useState } from 'react';
import { apiPost } from '../../api/client';
import { useToast } from '../shared/Toast';
import { useApi } from '../../hooks/useApi';
import { JsonViewer } from '../shared/JsonViewer';
import type { ScenarioInfo, ScenarioResult } from '../../api/types';

export function ScenariosPanel() {
  const toast = useToast();
  const scenarios = useApi<ScenarioInfo[]>('/api/v1/admin/simulate/scenarios');
  const [result, setResult] = useState<ScenarioResult | null>(null);
  const [running, setRunning] = useState<string | null>(null);
  const [customJson, setCustomJson] = useState('');
  const [runningCustom, setRunningCustom] = useState(false);

  const runScenario = async (name: string) => {
    setRunning(name);
    setResult(null);
    try {
      const res = await apiPost<ScenarioResult>(`/api/v1/admin/simulate/run`, { scenario: name });
      setResult(res);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Scenario failed');
    } finally {
      setRunning(null);
    }
  };

  const runCustom = async () => {
    setRunningCustom(true);
    setResult(null);
    try {
      const body = JSON.parse(customJson);
      const res = await apiPost<ScenarioResult>('/api/v1/admin/simulate/run-custom', body);
      setResult(res);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Invalid JSON or scenario failed');
    } finally {
      setRunningCustom(false);
    }
  };

  return (
    <div className="space-y-6">
      {/* Available Scenarios */}
      <div className="bg-gray-900 rounded-lg border border-gray-800 p-4">
        <h3 className="text-sm font-medium text-gray-300 mb-3">Available Scenarios</h3>
        {scenarios.isLoading && <p className="text-sm text-gray-500">Loading...</p>}
        {scenarios.error && <p className="text-sm text-red-400">{scenarios.error}</p>}
        {scenarios.data && (
          <div className="space-y-2">
            {scenarios.data.map(s => (
              <div key={s.name} className="flex items-center justify-between py-2 border-b border-gray-800 last:border-0">
                <div>
                  <p className="text-sm text-gray-200">{s.name}</p>
                  <p className="text-xs text-gray-500">{s.description}</p>
                </div>
                <button
                  onClick={() => runScenario(s.name)}
                  disabled={running === s.name}
                  className="px-3 py-1 text-xs rounded-md bg-blue-600 text-white hover:bg-blue-500 disabled:opacity-50 transition-colors"
                >
                  {running === s.name ? 'Running...' : 'Run'}
                </button>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Custom Scenario */}
      <div className="bg-gray-900 rounded-lg border border-gray-800 p-4">
        <h3 className="text-sm font-medium text-gray-300 mb-3">Custom Scenario</h3>
        <textarea
          value={customJson}
          onChange={e => setCustomJson(e.target.value)}
          placeholder='{"scenario": "...", "params": {...}}'
          rows={5}
          className="w-full px-3 py-2 bg-gray-800 border border-gray-700 rounded text-sm text-gray-200 font-mono focus:outline-none focus:border-blue-500 resize-y"
        />
        <button
          onClick={runCustom}
          disabled={runningCustom || !customJson.trim()}
          className="mt-2 px-3 py-1.5 text-sm rounded-md bg-blue-600 text-white hover:bg-blue-500 disabled:opacity-50 transition-colors"
        >
          {runningCustom ? 'Running...' : 'Run Custom'}
        </button>
      </div>

      {/* Result */}
      {result && (
        <div className="bg-gray-900 rounded-lg border border-gray-800 p-4">
          <h3 className="text-sm font-medium text-gray-300 mb-3">
            Result: <span className={result.success ? 'text-green-400' : 'text-red-400'}>{result.success ? 'Success' : 'Failed'}</span>
          </h3>
          <JsonViewer data={result} defaultExpanded />
        </div>
      )}
    </div>
  );
}

