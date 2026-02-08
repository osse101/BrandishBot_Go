import { useState } from 'react';
import { apiPost } from '../../api/client';
import { useToast } from '../shared/Toast';
import { ConfirmDialog } from '../shared/ConfirmDialog';
import { useApi } from '../../hooks/useApi';
import { JsonViewer } from '../shared/JsonViewer';

export function JobsPanel() {
  const toast = useToast();
  const [showResetConfirm, setShowResetConfirm] = useState(false);
  const resetStatus = useApi<unknown>('/api/v1/admin/jobs/reset-status');

  const [form, setForm] = useState({ platform: 'twitch', username: '', job_key: '', amount: '' });
  const [loading, setLoading] = useState(false);

  const handleAwardXP = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    try {
      await apiPost('/api/v1/admin/jobs/award-xp', {
        platform: form.platform,
        username: form.username,
        job_key: form.job_key,
        amount: Number(form.amount),
      });
      toast.success(`Awarded ${form.amount} XP to ${form.username}`);
      setForm(f => ({ ...f, username: '', amount: '' }));
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Failed to award XP');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="space-y-6">
      {/* Award XP */}
      <div className="bg-gray-900 rounded-lg border border-gray-800 p-4">
        <h3 className="text-sm font-medium text-gray-300 mb-3">Award XP</h3>
        <form onSubmit={handleAwardXP} className="flex items-end gap-2 flex-wrap">
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
          <div className="flex-1 min-w-[120px]">
            <label className="text-xs text-gray-500 block mb-1">Username</label>
            <input
              value={form.username}
              onChange={e => setForm(f => ({ ...f, username: e.target.value }))}
              className="w-full px-2 py-1.5 bg-gray-800 border border-gray-700 rounded text-sm text-gray-200 focus:outline-none focus:border-blue-500"
              placeholder="username"
            />
          </div>
          <div className="min-w-[100px]">
            <label className="text-xs text-gray-500 block mb-1">Job Key</label>
            <input
              value={form.job_key}
              onChange={e => setForm(f => ({ ...f, job_key: e.target.value }))}
              className="w-full px-2 py-1.5 bg-gray-800 border border-gray-700 rounded text-sm text-gray-200 focus:outline-none focus:border-blue-500"
              placeholder="e.g. miner"
            />
          </div>
          <div className="w-20">
            <label className="text-xs text-gray-500 block mb-1">Amount</label>
            <input
              type="number"
              value={form.amount}
              onChange={e => setForm(f => ({ ...f, amount: e.target.value }))}
              className="w-full px-2 py-1.5 bg-gray-800 border border-gray-700 rounded text-sm text-gray-200 focus:outline-none focus:border-blue-500"
              placeholder="100"
            />
          </div>
          <button
            type="submit"
            disabled={loading}
            className="px-3 py-1.5 text-sm rounded-md bg-blue-600 text-white hover:bg-blue-500 disabled:opacity-50 transition-colors"
          >
            {loading ? '...' : 'Award XP'}
          </button>
        </form>
      </div>

      {/* Daily Reset */}
      <div className="bg-gray-900 rounded-lg border border-gray-800 p-4">
        <h3 className="text-sm font-medium text-gray-300 mb-3">Daily XP Reset</h3>
        <div className="flex items-center gap-4">
          <button
            onClick={() => setShowResetConfirm(true)}
            className="px-3 py-1.5 text-sm rounded-md bg-red-600 text-white hover:bg-red-500 transition-colors"
          >
            Reset Daily XP
          </button>
          <div className="flex-1">
            {resetStatus.data != null && <JsonViewer data={resetStatus.data} />}
          </div>
        </div>
      </div>

      {showResetConfirm && (
        <ConfirmDialog
          title="Reset Daily XP"
          message="This will reset the daily XP cap for all users. Are you sure?"
          confirmLabel="Reset"
          onConfirm={async () => {
            setShowResetConfirm(false);
            try {
              await apiPost('/api/v1/admin/jobs/reset-daily-xp');
              toast.success('Daily XP reset');
              resetStatus.refetch();
            } catch (err) {
              toast.error(err instanceof Error ? err.message : 'Reset failed');
            }
          }}
          onCancel={() => setShowResetConfirm(false)}
        />
      )}
    </div>
  );
}

