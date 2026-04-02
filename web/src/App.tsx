import { BrowserRouter, Routes, Route } from 'react-router-dom'
import { useTheme } from '@/hooks/useTheme'
import AppLayout from '@/components/layout/AppLayout'
import InventoryPage from '@/pages/inventory/InventoryPage'
import ShoppingPage from '@/pages/shopping/ShoppingPage'
import HistoryPage from '@/pages/history/HistoryPage'
import DashboardPage from '@/pages/dashboard/DashboardPage'
import SettingsPage from '@/pages/settings/SettingsPage'

export default function App() {
  useTheme()

  return (
    <BrowserRouter>
      <Routes>
        <Route element={<AppLayout />}>
          <Route index element={<InventoryPage />} />
          <Route path="shopping" element={<ShoppingPage />} />
          <Route path="history" element={<HistoryPage />} />
          <Route path="dashboard" element={<DashboardPage />} />
          <Route path="settings" element={<SettingsPage />} />
        </Route>
      </Routes>
    </BrowserRouter>
  )
}
