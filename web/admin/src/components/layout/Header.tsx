import type { ConnectionStatus } from '../../hooks/useSSE';

interface Props {
  sseStatus: ConnectionStatus;
  onLogout: () => void;
}

const statusColors: Record<ConnectionStatus, string> = {
  connected: 'bg-green-400',
  connecting: 'bg-yellow-400 animate-pulse',
  disconnected: 'bg-gray-500',
};

const statusLabels: Record<ConnectionStatus, string> = {
  connected: 'SSE Connected',
  connecting: 'SSE Connecting...',
  disconnected: 'SSE Disconnected',
};

export function Header({ sseStatus, onLogout }: Props) {
  return (
    <header className="h-12 bg-gray-900 border-b border-gray-800 flex items-center justify-between px-4">
      <div />
      <div className="flex items-center gap-4">
        <div className="flex items-center gap-2 text-xs text-gray-400">
          <span className={`w-2 h-2 rounded-full ${statusColors[sseStatus]}`} />
          {statusLabels[sseStatus]}
        </div>
        <button
          onClick={onLogout}
          className="text-xs text-gray-400 hover:text-gray-200 transition-colors"
        >
          Logout
        </button>
      </div>
    </header>
  );
}
