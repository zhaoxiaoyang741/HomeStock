import { useEffect } from 'react'
import { BrowserRouter, Routes, Route } from 'react-router-dom'
import { useTheme } from '@/hooks/useTheme'
import { useAuthStore } from '@/store/authStore'
import AppLayout from '@/components/layout/AppLayout'
import ProtectedRoute from '@/components/auth/ProtectedRoute'
import LoginPage from '@/pages/auth/LoginPage'
import InventoryPage from '@/pages/inventory/InventoryPage'
import ShoppingPage from '@/pages/shopping/ShoppingPage'
import HistoryPage from '@/pages/history/HistoryPage'
import HomedPage from '@/pages/home/HomePage'
import SettingsPage from '@/pages/settings/SettingsPage'

export default function App() {
  useTheme()
  const initialize = useAuthStore((s) => s.initialize)

  useEffect(() => {
    initialize()
  }, [initialize])

  return (
    <BrowserRouter>
      <Routes>
        {/* Public auth pages */}
        <Route path="/login" element={<LoginPage />} />

        {/* Protected pages */}
        <Route element={<ProtectedRoute />}>
          <Route element={<AppLayout />}>
            <Route index element={<HomedPage />} />
            <Route path="home" element={<HomedPage />} />
            <Route path="inventory" element={<InventoryPage />} />
            <Route path="shopping" element={<ShoppingPage />} />
            <Route path="history" element={<HistoryPage />} />
            <Route path="settings" element={<SettingsPage />} />
          </Route>
        </Route>
      </Routes>
    </BrowserRouter>
  )
}
