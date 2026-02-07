import { useState, useEffect, useRef } from 'react';
import { useSSE } from '../hooks/useSSE';
import { formatTimestamp } from '../utils/format';
import { JsonViewer } from '../components/shared/JsonViewer';
import type { SSEEvent } from '../api/types';

const EVENT_CATEGORIES: Record<string, { color: string; types: string[] }> = {
  Gamble: { color: 'bg-amber-500/20 text-amber-400 border-amber-500/30', types: ['gamble.'] },
  Expedition: { color: 'bg-emerald-500/20 text-emerald-400 border-emerald-500/30', types: ['expedition.'] },
  Progression: { color: 'bg-purple-500/20 text-purple-400 border-purple-500/30', types: ['progression.'] },
  Jobs: { color: 'bg-blue-500/20 text-blue-400 border-blue-500/30', types: ['job.'] },
  Timeout: { color: 'bg-red-500/20 text-red-400 border-red-500/30', types: ['timeout.'] },
  Economy: { color: 'bg-yellow-500/20 text-yellow-400 border-yellow-500/30', types: ['item.', 'economy.'] },
};

function getCategoryColor(eventType: string): string {
  for (const cat of Object.values(EVENT_CATEGORIES)) {
    if (cat.types.some(prefix => eventType.startsWith(prefix))) {
      return cat.color;
    }
  }
  return 'bg-gray-500/20 text-gray-400 border-gray-500/30';
}

export function EventsPage() {
  const { events, status, clearEvents } = useSSE('/api/v1/events', true);
  const [filters, setFilters] = useState<Record<string, boolean>>(
    Object.fromEntries(Object.keys(EVENT_CATEGORIES).map(k => [k, true]))
  );
  const [autoScroll, setAutoScroll] = useState(true);
  const [expandedId, setExpandedId] = useState<string | null>(null);
  const listRef = useRef<HTMLDivElement>(null);

  const filteredEvents = events.filter(evt => {
    for (const [cat, cfg] of Object.entries(EVENT_CATEGORIES)) {
      if (cfg.types.some(prefix => evt.type.startsWith(prefix))) {
        return filters[cat];
      }
    }
    return true; // show uncategorized events
  });

  useEffect(() => {
    if (autoScroll && listRef.current) {
      listRef.current.scrollTop = listRef.current.scrollHeight;
    }
  }, [filteredEvents.length, autoScroll]);

  const handleScroll = () => {
    if (!listRef.current) return;
    const { scrollTop, scrollHeight, clientHeight } = listRef.current;
    const atBottom = scrollHeight - scrollTop - clientHeight < 50;
    if (!atBottom && autoScroll) setAutoScroll(false);
    if (atBottom && !autoScroll) setAutoScroll(true);
  };

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-xl font-semibold text-gray-100">Live Events</h2>
        <div className="flex items-center gap-3">
          <span className="text-xs text-gray-500">{events.length} events</span>
          <button
            onClick={() => setAutoScroll(!autoScroll)}
            className={`text-xs px-2 py-1 rounded ${autoScroll ? 'bg-blue-600/20 text-blue-400' : 'bg-gray-700 text-gray-400'}`}
          >
            Auto-scroll {autoScroll ? 'ON' : 'OFF'}
          </button>
          <button
            onClick={clearEvents}
            className="text-xs px-2 py-1 rounded bg-gray-700 text-gray-400 hover:text-gray-200 transition-colors"
          >
            Clear
          </button>
        </div>
      </div>

      {/* Filter bar */}
      <div className="flex flex-wrap gap-2 mb-4">
        {Object.keys(EVENT_CATEGORIES).map(cat => (
          <label key={cat} className="flex items-center gap-1.5 text-xs text-gray-400 cursor-pointer">
            <input
              type="checkbox"
              checked={filters[cat] ?? true}
              onChange={e => setFilters(f => ({ ...f, [cat]: e.target.checked }))}
              className="rounded border-gray-600"
            />
            {cat}
          </label>
        ))}
      </div>

      {/* Event list */}
      <div
        ref={listRef}
        onScroll={handleScroll}
        className="flex-1 overflow-auto space-y-1 min-h-0"
      >
        {filteredEvents.length === 0 && (
          <p className="text-sm text-gray-500 text-center py-8">
            {status === 'connected' ? 'Waiting for events...' : 'Not connected'}
          </p>
        )}
        {filteredEvents.map(evt => (
          <EventCard
            key={evt.id}
            event={evt}
            expanded={expandedId === evt.id}
            onToggle={() => setExpandedId(expandedId === evt.id ? null : evt.id)}
          />
        ))}
      </div>
    </div>
  );
}

function EventCard({ event, expanded, onToggle }: { event: SSEEvent; expanded: boolean; onToggle: () => void }) {
  return (
    <div
      className="bg-gray-900 rounded border border-gray-800 px-3 py-2 cursor-pointer hover:bg-gray-800/50 transition-colors"
      onClick={onToggle}
    >
      <div className="flex items-center gap-3">
        <span className="text-xs text-gray-500 font-mono w-20 shrink-0">
          {formatTimestamp(event.timestamp)}
        </span>
        <span className={`text-xs px-2 py-0.5 rounded-full border ${getCategoryColor(event.type)}`}>
          {event.type}
        </span>
      </div>
      {expanded && (
        <div className="mt-2" onClick={e => e.stopPropagation()}>
          <JsonViewer data={event.payload} defaultExpanded />
        </div>
      )}
    </div>
  );
}
