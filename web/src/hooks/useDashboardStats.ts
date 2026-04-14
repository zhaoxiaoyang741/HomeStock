import { useState, useEffect } from 'react'
import { materialApi } from '@/api/material'
import { categoryApi } from '@/api/category'
import { stockMovementApi } from '@/api/stockMovement'
import { getInventoryStatus } from '@/types/material'
import type { MaterialSummary } from '@/types/material'
import type { Category } from '@/types/category'
import type { StockMovement } from '@/types/stock'

export interface CategoryStat {
  name: string
  quantity: number
  count: number
}

export interface DashboardStats {
  totalMaterials: number
  totalQuantity: number
  expiringCount: number
  expiredCount: number
  normalCount: number
  byCategory: CategoryStat[]
  recentMovements: StockMovement[]
}

export function useDashboardStats(remindDays: number = 3) {
  const [stats, setStats] = useState<DashboardStats | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false

    async function fetchStats() {
      setLoading(true)
      setError(null)
      try {
        const [materialsRes, categoriesRes, movementsRes] = await Promise.all([
          materialApi.list(),
          categoryApi.list(),
          stockMovementApi.list(),
        ])

        if (cancelled) return

        const materials: MaterialSummary[] = materialsRes.materials
        const categories: Category[] = categoriesRes.categories
        const recentMovements = movementsRes.movements.slice(0, 10)

        const categoryMap = new Map<string, Category>()
        for (const cat of categories) {
          categoryMap.set(cat.id, cat)
        }

        let totalQuantity = 0
        let expiringCount = 0
        let expiredCount = 0
        let normalCount = 0

        const catStats = new Map<string, { name: string; quantity: number; count: number }>()

        for (const m of materials) {
          const status = getInventoryStatus(m.nearest_expire_at, remindDays)
          totalQuantity += m.total_quantity
          if (status === 'expiring') expiringCount++
          else if (status === 'expired') expiredCount++
          else normalCount++

          const catId = m.category_id || '__uncategorized__'
          const catName = m.category?.name ?? (m.category_id ? m.category_id : '未分类')
          const existing = catStats.get(catId) ?? { name: catName, quantity: 0, count: 0 }
          existing.quantity += m.total_quantity
          existing.count += 1
          catStats.set(catId, existing)
        }

        const byCategory: CategoryStat[] = Array.from(catStats.values())
          .sort((a, b) => b.quantity - a.quantity)

        setStats({
          totalMaterials: materials.length,
          totalQuantity,
          expiringCount,
          expiredCount,
          normalCount,
          byCategory,
          recentMovements,
        })
      } catch (e) {
        if (!cancelled) {
          setError(e instanceof Error ? e.message : 'Unknown error')
        }
      } finally {
        if (!cancelled) setLoading(false)
      }
    }

    fetchStats()
    return () => { cancelled = true }
  }, [remindDays])

  return { stats, loading, error }
}
