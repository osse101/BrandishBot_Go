import { useState } from 'react';

interface Props {
  data: unknown;
  defaultExpanded?: boolean;
}

export function JsonViewer({ data, defaultExpanded = false }: Props) {
  const [expanded, setExpanded] = useState(defaultExpanded);
  const json = JSON.stringify(data, null, 2);

  if (!expanded) {
    const preview = JSON.stringify(data);
    const truncated = preview.length > 80 ? preview.slice(0, 80) + '...' : preview;
    return (
      <button
        onClick={() => setExpanded(true)}
        className="text-left font-mono text-xs text-gray-400 hover:text-gray-200 transition-colors"
      >
        {truncated}
      </button>
    );
  }

  return (
    <div className="relative">
      <button
        onClick={() => setExpanded(false)}
        className="absolute top-1 right-1 text-xs text-gray-500 hover:text-gray-300"
      >
        collapse
      </button>
      <pre className="bg-gray-900 rounded p-3 text-xs font-mono text-gray-300 overflow-x-auto max-h-64 overflow-y-auto">
        {json}
      </pre>
    </div>
  );
}
