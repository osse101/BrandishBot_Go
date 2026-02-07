interface Column<T> {
  key: string;
  header: string;
  render?: (row: T) => React.ReactNode;
}

interface Props<T> {
  columns: Column<T>[];
  data: T[];
  keyField: string;
  emptyMessage?: string;
}

export function DataTable<T extends Record<string, unknown>>({ columns, data, keyField, emptyMessage = 'No data' }: Props<T>) {
  if (data.length === 0) {
    return <p className="text-gray-500 text-sm py-4 text-center">{emptyMessage}</p>;
  }

  return (
    <div className="overflow-x-auto">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-gray-700">
            {columns.map(col => (
              <th key={col.key} className="text-left py-2 px-3 text-gray-400 font-medium">
                {col.header}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {data.map(row => (
            <tr key={String(row[keyField])} className="border-b border-gray-800 hover:bg-gray-800/50">
              {columns.map(col => (
                <td key={col.key} className="py-2 px-3 text-gray-300">
                  {col.render ? col.render(row) : String(row[col.key] ?? '')}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
