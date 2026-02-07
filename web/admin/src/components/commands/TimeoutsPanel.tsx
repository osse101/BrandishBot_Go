import { useState } from 'react';
import { apiPost } from '../../api/client';
import { useToast } from '../shared/Toast';

export function TimeoutsPanel() {
  const toast = useToast();
  const [form, setForm] = useState({ platform: 'twitch', username: '' });
  const [loading, setLoading] = useState(false);

  const handleClear = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    try {
      await apiPost('/api/v1/admin/timeout/clear', {
        platform: form.platform,
        username: form.username,
      });
      toast.success(`Timeout cleared for ${form.username}`);
      setForm(f => ({ ...f, username: '' }));
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Failed to clear timeout');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="space-y-6">
      <div className="bg-gray-900 rounded-lg border border-gray-800 p-4">
        <h3 className="text-sm font-medium text-gray-300 mb-3">Clear User Timeout</h3>
        <form onSubmit={handleClear} className="flex items-end gap-2">
          <div>
            <label className="text-xs text-gray-500 block mb-1">Platform</label>
            <select
              value={form.platform}
              onChange={e => setForm(f => ({ ...f, platform: e.target.value }))}
              className="px-2 py-1.5 bg-gray-800 border border-gray-700 rounded text-sm text-gray-200 focus:outline-none focus:border-blue-500"
            >
              <option value="twitch">Twitch</option>
              <option value="discord">Discord</option>
              <option value="youtube">YouTube</option>
            </select>
          </div>
          <div className="flex-1">
            <label className="text-xs text-gray-500 block mb-1">Username</label>
            <input
              value={form.username}
              onChange={e => setForm(f => ({ ...f, username: e.target.value }))}
              className="w-full px-2 py-1.5 bg-gray-800 border border-gray-700 rounded text-sm text-gray-200 focus:outline-none focus:border-blue-500"
              placeholder="username"
            />
          </div>
          <button
            type="submit"
            disabled={loading || !form.username.trim()}
            className="px-3 py-1.5 text-sm rounded-md bg-blue-600 text-white hover:bg-blue-500 disabled:opacity-50 transition-colors"
          >
            {loading ? '...' : 'Clear Timeout'}
          </button>
        </form>
      </div>
    </div>
  );
}
