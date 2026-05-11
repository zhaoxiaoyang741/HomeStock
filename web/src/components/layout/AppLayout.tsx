import { Outlet } from 'react-router-dom'
import Sidebar from './Sidebar'
import Header from './Header'
import { cn } from '@/lib/utils'
import { useAppStore } from '@/store/appStore'

export default function AppLayout() {
  const collapsed = useAppStore((s) => s.collapsed)

  return (
    <div className="flex h-screen overflow-hidden bg-background flex-1">
      <Sidebar />
      <div className={cn(
        'flex-1 flex min-h-0 flex-col transition-[margin-left] duration-300 ease-in-out',
        collapsed ? 'ml-16' : 'ml-50'
      )}>
        <Header />
        <main className="flex-1 min-h-0 mt-16 overflow-hidden p-6">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
