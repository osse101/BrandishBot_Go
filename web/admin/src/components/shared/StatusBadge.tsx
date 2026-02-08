interface Props {
  status: 'ok' | 'error' | 'warning' | 'unknown';
  label: string;
}

const colors: Record<Props['status'], string> = {
  ok: 'bg-green-500/20 text-green-400 border-green-500/30',
  error: 'bg-red-500/20 text-red-400 border-red-500/30',
  warning: 'bg-yellow-500/20 text-yellow-400 border-yellow-500/30',
  unknown: 'bg-gray-500/20 text-gray-400 border-gray-500/30',
};

export function StatusBadge({ status, label }: Props) {
  return (
    <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium border ${colors[status]}`}>
      <span className={`w-1.5 h-1.5 rounded-full mr-1.5 ${status === 'ok' ? 'bg-green-400' : status === 'error' ? 'bg-red-400' : status === 'warning' ? 'bg-yellow-400' : 'bg-gray-400'}`} />
      {label}
    </span>
  );
}
