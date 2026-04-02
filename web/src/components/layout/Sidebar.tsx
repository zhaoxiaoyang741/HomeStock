import { NavLink } from 'react-router-dom'
import {
  Package,
  ShoppingCart,
  ClipboardList,
  BarChart2,
  Settings,
  LogOut,
  Warehouse,
} from 'lucide-react'
import { cn } from '@/lib/utils'

const NAV_ITEMS = [
  { path: '/',          icon: Package,       label: '物料库存' },
  { path: '/shopping',  icon: ShoppingCart,  label: '购物列表' },
  { path: '/history',   icon: ClipboardList, label: '使用记录' },
  { path: '/dashboard', icon: BarChart2,     label: '数据展板' },
  { path: '/settings',  icon: Settings,      label: '系统设置' },
]

export default function Sidebar() {
  return (
    <aside className="fixed inset-y-0 left-0 w-[260px] flex flex-col bg-surface-container-low border-r border-outline-variant z-20">
      {/* Logo */}
      <div className="flex items-center gap-3 px-6 h-16 border-b border-outline-variant shrink-0">
        <div className="w-8 h-8 rounded-lg bg-gradient-primary flex items-center justify-center">
          <Warehouse className="w-5 h-5 text-on-primary" />
        </div>
        <div>
          <p className="text-sm font-semibold text-on-surface leading-tight">Amenity Home</p>
          <p className="text-xs text-on-surface-variant leading-tight">物料管理</p>
        </div>
      </div>

      {/* Navigation */}
      <nav className="flex-1 px-3 py-4 space-y-1 overflow-y-auto">
        {NAV_ITEMS.map(({ path, icon: Icon, label }) => (
          <NavLink
            key={path}
            to={path}
            end={path === '/'}
            className={({ isActive }) =>
              cn(
                'flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-colors',
                isActive
                  ? 'bg-primary/10 text-primary'
                  : 'text-on-surface-variant hover:bg-surface-container hover:text-on-surface'
              )
            }
          >
            <Icon className="w-5 h-5 shrink-0" />
            {label}
          </NavLink>
        ))}
      </nav>

      {/* Footer */}
      <div className="px-3 py-4 border-t border-outline-variant shrink-0">
        <button className="flex items-center gap-3 w-full px-3 py-2.5 rounded-lg text-sm font-medium text-on-surface-variant hover:bg-surface-container hover:text-on-surface transition-colors">
          <LogOut className="w-5 h-5 shrink-0" />
          退出登录
        </button>
      </div>
    </aside>
  )
}
