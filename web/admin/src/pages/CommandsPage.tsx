import { useState } from 'react';
import { ProgressionPanel } from '../components/commands/ProgressionPanel';
import { JobsPanel } from '../components/commands/JobsPanel';
import { CachePanel } from '../components/commands/CachePanel';
import { ScenariosPanel } from '../components/commands/ScenariosPanel';
import { TimeoutsPanel } from '../components/commands/TimeoutsPanel';

const tabs = ['Progression', 'Jobs', 'Cache', 'Scenarios', 'Timeouts'] as const;
type Tab = typeof tabs[number];

export function CommandsPage() {
  const [activeTab, setActiveTab] = useState<Tab>('Progression');

  return (
    <div className="space-y-4">
      <h2 className="text-xl font-semibold text-gray-100">Admin Commands</h2>

      <div className="flex gap-1 border-b border-gray-800">
        {tabs.map(tab => (
          <button
            key={tab}
            onClick={() => setActiveTab(tab)}
            className={`px-4 py-2 text-sm transition-colors ${
              activeTab === tab
                ? 'text-blue-400 border-b-2 border-blue-400'
                : 'text-gray-400 hover:text-gray-200'
            }`}
          >
            {tab}
          </button>
        ))}
      </div>

      <div className="mt-4">
        {activeTab === 'Progression' && <ProgressionPanel />}
        {activeTab === 'Jobs' && <JobsPanel />}
        {activeTab === 'Cache' && <CachePanel />}
        {activeTab === 'Scenarios' && <ScenariosPanel />}
        {activeTab === 'Timeouts' && <TimeoutsPanel />}
      </div>
    </div>
  );
}
