import { useState } from 'react';

interface Props {
  onLogin: (key: string) => Promise<void>;
  isLoading: boolean;
  error: string | null;
}

export function LoginPage({ onLogin, isLoading, error }: Props) {
  const [key, setKey] = useState('');

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (key.trim()) onLogin(key.trim());
  };

  return (
    <div className="flex items-center justify-center min-h-screen bg-gray-950">
      <div className="bg-gray-900 rounded-lg p-8 w-full max-w-sm border border-gray-800 shadow-xl">
        <h1 className="text-xl font-bold text-gray-100 mb-1">BrandishBot Admin</h1>
        <p className="text-sm text-gray-500 mb-6">Enter your API key to continue</p>

        <form onSubmit={handleSubmit}>
          <input
            type="password"
            value={key}
            onChange={e => setKey(e.target.value)}
            placeholder="API Key"
            className="w-full px-3 py-2 bg-gray-800 border border-gray-700 rounded-md text-gray-200 text-sm focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500 placeholder-gray-500"
            autoFocus
            disabled={isLoading}
          />

          {error && (
            <p className="mt-2 text-sm text-red-400">{error}</p>
          )}

          <button
            type="submit"
            disabled={isLoading || !key.trim()}
            className="mt-4 w-full px-4 py-2 bg-blue-600 text-white text-sm font-medium rounded-md hover:bg-blue-500 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            {isLoading ? 'Authenticating...' : 'Login'}
          </button>
        </form>
      </div>
    </div>
  );
}
