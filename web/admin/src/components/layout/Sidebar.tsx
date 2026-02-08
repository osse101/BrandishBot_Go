import { NavLink } from 'react-router-dom';

const navItems = [
  { to: '/admin/', label: 'Health', icon: '♥' },
  { to: '/admin/commands', label: 'Commands', icon: '⌘' },
  { to: '/admin/events', label: 'Events', icon: '⚡' },
  { to: '/admin/users', label: 'Users', icon: '👤' },
];

export function Sidebar() {
  return (
    <aside className="w-56 bg-gray-900 border-r border-gray-800 flex flex-col">
      <div className="p-4 border-b border-gray-800">
        <h1 className="text-lg font-bold text-gray-100">BrandishBot</h1>
        <p className="text-xs text-gray-500">Admin Dashboard</p>
      </div>
      <nav className="flex-1 p-2">
        {navItems.map(item => (
          <NavLink
            key={item.to}
            to={item.to}
            end={item.to === '/admin/'}
            className={({ isActive }) =>
              `flex items-center gap-3 px-3 py-2 rounded-md text-sm transition-colors ${
                isActive
                  ? 'bg-blue-600/20 text-blue-400'
                  : 'text-gray-400 hover:bg-gray-800 hover:text-gray-200'
              }`
            }
          >
            <span className="text-base">{item.icon}</span>
            {item.label}
          </NavLink>
        ))}
      </nav>
    </aside>
  );
}
