import { useState } from 'react';
import { apiPost } from '../../api/client';
import { useToast } from '../shared/Toast';
import { ConfirmDialog } from '../shared/ConfirmDialog';
import { JsonViewer } from '../shared/JsonViewer';
import type { VotingSession } from '../../api/types';
import { useApi } from '../../hooks/useApi';

export function ProgressionPanel() {
  const toast = useToast();
  const session = useApi<VotingSession>('/api/v1/progression/session');
  const [showResetConfirm, setShowResetConfirm] = useState(false);

  return (
    <div className="space-y-6">
      {/* Current Session */}
      <Section title="Current Voting Session">
        {session.isLoading && <p className="text-sm text-gray-500">Loading...</p>}
        {session.data && <JsonViewer data={session.data} defaultExpanded />}
        {session.error && <p className="text-sm text-red-400">{session.error}</p>}
      </Section>

      {/* Voting Controls */}
      <Section title="Voting Controls">
        <div className="flex flex-wrap gap-2">
          <ActionButton label="Start Voting" onClick={async () => {
            await apiPost('/api/v1/progression/admin/start-voting');
            toast.success('Voting started');
            session.refetch();
          }} />
          <ActionButton label="End Voting (Freeze)" onClick={async () => {
            await apiPost('/api/v1/progression/admin/end-voting');
            toast.success('Voting frozen');
            session.refetch();
          }} />
          <ActionButton label="Force End Voting" variant="warning" onClick={async () => {
            await apiPost('/api/v1/progression/admin/force-end-voting');
            toast.success('Voting force-ended');
            session.refetch();
          }} />
          <ActionButton label="Instant Unlock Leader" onClick={async () => {
            await apiPost('/api/v1/progression/admin/instant-unlock');
            toast.success('Leader node unlocked');
            session.refetch();
          }} />
        </div>
      </Section>

      {/* Unlock/Relock Node */}
      <Section title="Node Management">
        <FormRow label="Unlock Node" fields={['node_index']} onSubmit={async (vals) => {
          await apiPost('/api/v1/progression/admin/unlock', { node_index: Number(vals.node_index) });
          toast.success('Node unlocked');
        }} />
        <FormRow label="Relock Node" fields={['node_index']} onSubmit={async (vals) => {
          await apiPost('/api/v1/progression/admin/relock', { node_index: Number(vals.node_index) });
          toast.success('Node relocked');
        }} />
        <FormRow label="Add Contribution" fields={['platform', 'username', 'amount']} onSubmit={async (vals) => {
          await apiPost('/api/v1/progression/admin/contribution', {
            platform: vals.platform,
            username: vals.username,
            amount: Number(vals.amount),
          });
          toast.success('Contribution added');
        }} />
      </Section>

      {/* Dangerous Actions */}
      <Section title="Dangerous Actions">
        <ActionButton label="Reset Tree" variant="danger" onClick={() => setShowResetConfirm(true)} />
        <ActionButton label="Unlock All Nodes" variant="danger" onClick={async () => {
          await apiPost('/api/v1/progression/admin/unlock-all');
          toast.success('All nodes unlocked');
        }} />
      </Section>

      {showResetConfirm && (
        <ConfirmDialog
          title="Reset Progression Tree"
          message="This will relock all progression nodes and reset all voting data. This action cannot be undone."
          confirmLabel="Reset Tree"
          onConfirm={async () => {
            setShowResetConfirm(false);
            await apiPost('/api/v1/progression/admin/reset');
            toast.success('Progression tree reset');
            session.refetch();
          }}
          onCancel={() => setShowResetConfirm(false)}
        />
      )}
    </div>
  );
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="bg-gray-900 rounded-lg border border-gray-800 p-4">
      <h3 className="text-sm font-medium text-gray-300 mb-3">{title}</h3>
      {children}
    </div>
  );
}

function ActionButton({ label, variant = 'default', onClick }: {
  label: string;
  variant?: 'default' | 'warning' | 'danger';
  onClick: () => void | Promise<void>;
}) {
  const [loading, setLoading] = useState(false);
  const toast = useToast();

  const colors = {
    default: 'bg-blue-600 hover:bg-blue-500',
    warning: 'bg-yellow-600 hover:bg-yellow-500',
    danger: 'bg-red-600 hover:bg-red-500',
  };

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
      className={`px-3 py-1.5 text-sm rounded-md text-white disabled:opacity-50 transition-colors ${colors[variant]}`}
    >
      {loading ? '...' : label}
    </button>
  );
}

function FormRow({ label, fields, onSubmit }: {
  label: string;
  fields: string[];
  onSubmit: (values: Record<string, string>) => Promise<void>;
}) {
  const [values, setValues] = useState<Record<string, string>>(
    Object.fromEntries(fields.map(f => [f, '']))
  );
  const [loading, setLoading] = useState(false);
  const toast = useToast();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    try {
      await onSubmit(values);
      setValues(Object.fromEntries(fields.map(f => [f, ''])));
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Action failed');
    } finally {
      setLoading(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="flex items-end gap-2 mb-2">
      {fields.map(field => (
        <div key={field} className="flex-1">
          <label className="text-xs text-gray-500 block mb-1">{field}</label>
          <input
            value={values[field]}
            onChange={e => setValues(v => ({ ...v, [field]: e.target.value }))}
            className="w-full px-2 py-1.5 bg-gray-800 border border-gray-700 rounded text-sm text-gray-200 focus:outline-none focus:border-blue-500"
            placeholder={field}
          />
        </div>
      ))}
      <button
        type="submit"
        disabled={loading}
        className="px-3 py-1.5 text-sm rounded-md bg-blue-600 text-white hover:bg-blue-500 disabled:opacity-50 transition-colors whitespace-nowrap"
      >
        {loading ? '...' : label}
      </button>
    </form>
  );
}

