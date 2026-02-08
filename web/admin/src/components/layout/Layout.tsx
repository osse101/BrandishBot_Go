import { Outlet } from 'react-router-dom';
import { Sidebar } from './Sidebar';
import { Header } from './Header';
import type { ConnectionStatus } from '../../hooks/useSSE';

interface Props {
  sseStatus: ConnectionStatus;
  onLogout: () => void;
}

export function Layout({ sseStatus, onLogout }: Props) {
  return (
    <div className="flex h-screen">
      <Sidebar />
      <div className="flex-1 flex flex-col overflow-hidden">
        <Header sseStatus={sseStatus} onLogout={onLogout} />
        <main className="flex-1 overflow-auto p-6">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
