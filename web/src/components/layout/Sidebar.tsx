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
import { useAppStore } from '@/store/appStore'

const NAV_ITEMS = [
  { path: '/',          icon: Package,       label: '物料库存' },
  { path: '/shopping',  icon: ShoppingCart,  label: '购物列表' },
  { path: '/history',   icon: ClipboardList, label: '使用记录' },
  { path: '/dashboard', icon: BarChart2,     label: '数据展板' },
  { path: '/settings',  icon: Settings,      label: '系统设置' },
]

export default function Sidebar() {
  const collapsed = useAppStore((s) => s.collapsed)

  return (
    <aside
      className={cn(
        'fixed inset-y-0 left-0 flex flex-col bg-surface-container-low border-r border-outline-variant z-20',
        'sidebar-transition overflow-hidden',
        collapsed ? 'w-16' : 'w-50'
      )}
    >
      {/* Logo */}
      <div className="flex items-center h-16 border-b border-outline-variant shrink-0 px-4 gap-3">
        <div className="w-8 h-8 rounded-lg bg-gradient-primary flex items-center justify-center shrink-0">
          <Warehouse className="w-5 h-5 text-on-primary" />
        </div>
        <div className={cn('overflow-hidden sidebar-fade flex-1 min-w-0', collapsed ? 'opacity-0 w-0' : 'opacity-100 w-auto')}>
          <p className="text-sm font-semibold text-on-surface leading-tight whitespace-nowrap">Amenity Home</p>
          <p className="text-xs text-on-surface-variant leading-tight whitespace-nowrap">物料管理</p>
        </div>
      </div>

      {/* Navigation */}
      <nav className="flex-1 px-2 py-4 space-y-1 overflow-y-auto overflow-x-hidden">
        {NAV_ITEMS.map(({ path, icon: Icon, label }) => (
          <NavLink
            key={path}
            to={path}
            end={path === '/'}
            className={({ isActive }) =>
              cn(
                'flex items-center px-3 py-2.5 rounded-lg text-sm font-medium transition-colors ',
                isActive
                  ? 'bg-primary/10 text-primary'
                  : 'text-on-surface-variant hover:bg-surface-container hover:text-on-surface'
              )
            }
            title={ collapsed ? label : undefined}
          >
            <Icon className="w-5 h-5 shrink-0" />
            <span className={cn('overflow-hidden whitespace-nowrap transition-[opacity,width] duration-300 ease-in-out', collapsed ? 'opacity-0 w-0' : 'opacity-100 ml-4 w-auto')}>
              {label}
            </span>
          </NavLink>
        ))}
      </nav>

      {/* Footer */}
      <div className="px-2 py-4 border-t border-outline-variant shrink-0">
        <button
          className={cn(
            'flex items-center cursor-pointer gap-3 w-full px-3 py-2.5 rounded-lg text-sm font-medium text-on-surface-variant hover:bg-surface-container hover:text-on-surface transition-colors',
            collapsed ? 'justify-center' : ''
          )}
          title={collapsed ? '退出登录' : undefined}
        >
          <LogOut className="w-5 h-5 shrink-0" />
          <span className={cn('overflow-hidden whitespace-nowrap transition-[opacity,width] duration-300 ease-in-out', collapsed ? 'opacity-0 w-0' : 'opacity-100 w-auto')}>
            退出登录
          </span>
        </button>
      </div>
    </aside>
  )
}
