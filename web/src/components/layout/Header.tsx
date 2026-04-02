import { Bell, Moon, PanelLeftClose, PanelLeftOpen, RefreshCw, Search } from 'lucide-react'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Input } from '@/components/ui/input'
import { cn } from '@/lib/utils'
import { useAppStore } from '@/store/appStore'

export default function Header() {
  const { collapsed, toggleCollapsed } = useAppStore()

  return (
    <header className={cn(
      'fixed top-0 right-0 h-16 flex items-center gap-4 px-4 bg-surface/80 glass-effect border-b border-outline-variant z-10',
      'transition-[left] duration-300 ease-in-out',
      collapsed ? 'left-16' : 'left-50'
    )}>
      {/* Sidebar toggle */}
      <button
        onClick={toggleCollapsed}
        className="p-2 rounded-lg text-on-surface-variant hover:bg-surface-container hover:text-on-surface transition-colors shrink-0"
        aria-label={collapsed ? '展开侧边栏' : '收起侧边栏'}
      >
        {collapsed
          ? <PanelLeftOpen className="w-5 h-5" />
          : <PanelLeftClose className="w-5 h-5" />
        }
      </button>

      {/* Search */}
      <div className="relative w-72">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-on-surface-variant" />
        <Input className="pl-9 bg-surface-container-low border-outline-variant" placeholder="搜索物料..." />
      </div>

      {/* Actions */}
      <div className="flex items-center gap-2 ml-auto">
        <button className="p-2 rounded-lg text-on-surface-variant hover:bg-surface-container transition-colors" aria-label="切换暗色模式">
          <Moon className="w-5 h-5" />
        </button>
        <button className="p-2 rounded-lg text-on-surface-variant hover:bg-surface-container transition-colors" aria-label="同步">
          <RefreshCw className="w-5 h-5" />
        </button>
        <button className="p-2 rounded-lg text-on-surface-variant hover:bg-surface-container transition-colors" aria-label="通知">
          <Bell className="w-5 h-5" />
        </button>
        <Avatar className="w-8 h-8 ml-1 cursor-pointer">
          <AvatarImage src="" alt="用户头像" />
          <AvatarFallback className="bg-primary text-on-primary text-xs">用</AvatarFallback>
        </Avatar>
      </div>
    </header>
  )
}
