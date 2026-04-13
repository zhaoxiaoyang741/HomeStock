import { BrowserRouter, Routes, Route } from 'react-router-dom'
import { useTheme } from '@/hooks/useTheme'
import AppLayout from '@/components/layout/AppLayout'
import InventoryPage from '@/pages/inventory/InventoryPage'
import ShoppingPage from '@/pages/shopping/ShoppingPage'
import HistoryPage from '@/pages/history/HistoryPage'
import HomedPage from '@/pages/home/HomePage'
import SettingsPage from '@/pages/settings/SettingsPage'
import { SystemSettingsProvider } from '@/providers/SystemSettingsProvider'

export default function App() {
  useTheme()

  return (
    <BrowserRouter>
      <SystemSettingsProvider>
        <Routes>
          <Route element={<AppLayout />}>
            <Route index element={<HomedPage />} />
            <Route path="home" element={<HomedPage />} />
            <Route path="inventory" element={<InventoryPage />} />
            <Route path="shopping" element={<ShoppingPage />} />
            <Route path="history" element={<HistoryPage />} />
            <Route path="settings" element={<SettingsPage />} />
          </Route>
        </Routes>
      </SystemSettingsProvider>
    </BrowserRouter>
  )
}
