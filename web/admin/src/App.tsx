import { Routes, Route } from 'react-router-dom';
import { useAuth } from './hooks/useAuth';
import { useSSE } from './hooks/useSSE';
import { Layout } from './components/layout/Layout';
import { LoginPage } from './pages/LoginPage';
import { HealthPage } from './pages/HealthPage';
import { CommandsPage } from './pages/CommandsPage';
import { EventsPage } from './pages/EventsPage';
import { UsersPage } from './pages/UsersPage';

export function App() {
  const { isAuthenticated, isLoading, error, login, logout } = useAuth();
  const { status: sseStatus } = useSSE('/api/v1/events', isAuthenticated);

  if (!isAuthenticated) {
    return <LoginPage onLogin={login} isLoading={isLoading} error={error} />;
  }

  return (
    <Routes>
      <Route element={<Layout sseStatus={sseStatus} onLogout={logout} />}>
        <Route path="/admin/" element={<HealthPage />} />
        <Route path="/admin/commands" element={<CommandsPage />} />
        <Route path="/admin/events" element={<EventsPage />} />
        <Route path="/admin/users" element={<UsersPage />} />
      </Route>
    </Routes>
  );
}
