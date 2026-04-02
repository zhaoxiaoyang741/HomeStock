import { Outlet } from 'react-router-dom'
import Sidebar from './Sidebar'
import Header from './Header'
import { cn } from '@/lib/utils'
import { useAppStore } from '@/store/appStore'

export default function AppLayout() {
  const collapsed = useAppStore((s) => s.collapsed)

  return (
    <div className="flex min-h-screen bg-background flex-1">
      <Sidebar />
      <div className={cn(
        'flex-1 flex flex-col transition-[margin-left] duration-300 ease-in-out',
        collapsed ? 'ml-16' : 'ml-50'
      )}>
        <Header />
        <main className="flex-1 mt-16 p-6 overflow-auto">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
