import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import ReactECharts from 'echarts-for-react'
import { Package, AlertTriangle, XCircle, Layers } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { cn } from '@/lib/utils'
import { formatDateTime } from '@/lib/format'
import { useDashboardStats } from '@/hooks/useDashboardStats'
import { useTheme } from '@/hooks/useTheme'
import type { StockMovementType } from '@/types/stock'
import { useSystemSettings } from '@/hooks/useSystemSettings'

function getCssVar(name: string): string {
  return getComputedStyle(document.documentElement).getPropertyValue(name).trim()
}

function useChartColors() {
  const { resolvedTheme } = useTheme()
  return useMemo(() => ({
    primary: getCssVar('--color-primary'),
    tertiary: getCssVar('--color-tertiary'),
    error: getCssVar('--color-error'),
    onSurface: getCssVar('--color-on-surface'),
    onSurfaceVariant: getCssVar('--color-on-surface-variant'),
    surfaceContainerHigh: getCssVar('--color-surface-container-high'),
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }), [resolvedTheme])
}

interface StatCardProps {
  label: string
  value: number | string
  icon: React.ReactNode
  variant?: 'default' | 'warning' | 'error'
}

function StatCard({ label, value, icon, variant = 'default' }: StatCardProps) {
  return (
    <Card className="bg-surface-container-low border-outline-variant">
      <CardContent className="flex items-center gap-4 pt-6">
        <div
          className={cn(
            'flex h-12 w-12 items-center justify-center rounded-xl shrink-0',
            variant === 'default' && 'bg-primary/10 text-primary',
            variant === 'warning' && 'bg-tertiary-container/40 text-tertiary',
            variant === 'error' && 'bg-error-container/40 text-error',
          )}
        >
          {icon}
        </div>
        <div className="min-w-0">
          <p className="text-2xl font-bold text-on-surface tabular-nums">{value}</p>
          <p className="text-sm text-on-surface-variant truncate">{label}</p>
        </div>
      </CardContent>
    </Card>
  )
}

function movementBadgeVariant(type: StockMovementType): 'default' | 'secondary' | 'destructive' | 'outline' {
  switch (type) {
    case 'inbound': return 'default'
    case 'consume': return 'secondary'
    case 'adjustment': return 'outline'
    case 'void': return 'destructive'
  }
}

export default function HomePage() {
  const { t } = useTranslation('dashboard')
  const { settings } = useSystemSettings()
  const remindDays = settings?.reminder?.remind_days ?? 3
  const { stats, loading, error } = useDashboardStats(remindDays)
  const colors = useChartColors()

  const movementTypeLabel = (type: StockMovementType) => {
    const map: Record<StockMovementType, string> = {
      inbound: t('movement_type_inbound'),
      consume: t('movement_type_consume'),
      adjustment: t('movement_type_adjustment'),
      void: t('movement_type_void'),
    }
    return map[type] ?? type
  }

  const barOption = useMemo(() => {
    if (!stats) return {}
    const sorted = [...stats.byCategory].reverse()
    return {
      tooltip: {
        trigger: 'axis',
        axisPointer: { type: 'shadow' },
        backgroundColor: getCssVar('--color-surface-container-high'),
        borderColor: getCssVar('--color-outline-variant'),
        textStyle: { color: getCssVar('--color-on-surface') },
      },
      grid: { left: '3%', right: '8%', bottom: '3%', containLabel: true },
      xAxis: {
        type: 'value',
        axisLabel: { color: colors.onSurfaceVariant },
        splitLine: { lineStyle: { color: colors.surfaceContainerHigh } },
      },
      yAxis: {
        type: 'category',
        data: sorted.map((c) => c.name),
        axisLabel: { color: colors.onSurfaceVariant },
      },
      series: [
        {
          name: t('total_quantity'),
          type: 'bar',
          data: sorted.map((c) => c.quantity),
          itemStyle: { color: colors.primary, borderRadius: [0, 4, 4, 0] },
          label: { show: true, position: 'right', color: colors.onSurfaceVariant, fontSize: 12 },
        },
      ],
    }
  }, [stats, colors, t])

  const pieOption = useMemo(() => {
    if (!stats) return {}
    return {
      tooltip: {
        trigger: 'item',
        backgroundColor: getCssVar('--color-surface-container-high'),
        borderColor: getCssVar('--color-outline-variant'),
        textStyle: { color: getCssVar('--color-on-surface') },
        formatter: '{b}: {c} ({d}%)',
      },
      legend: {
        orient: 'horizontal',
        bottom: 0,
        textStyle: { color: colors.onSurface },
      },
      series: [
        {
          type: 'pie',
          radius: ['40%', '70%'],
          center: ['50%', '45%'],
          avoidLabelOverlap: false,
          label: { show: false },
          emphasis: { label: { show: true, fontSize: 14, fontWeight: 'bold' } },
          data: [
            { value: stats.normalCount, name: t('status_normal'), itemStyle: { color: colors.primary } },
            { value: stats.expiringCount, name: t('status_expiring'), itemStyle: { color: colors.tertiary } },
            { value: stats.expiredCount, name: t('status_expired'), itemStyle: { color: colors.error } },
          ],
        },
      ],
    }
  }, [stats, colors, t])

  if (loading) {
    return (
      <div className="flex h-64 items-center justify-center text-on-surface-variant">
        {t('loading')}
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex h-64 items-center justify-center text-error">
        {t('error')}: {error}
      </div>
    )
  }

  if (!stats) return null

  const barHeight = Math.max(240, stats.byCategory.length * 44 + 48)

  return (
    <div className="space-y-6">
      {/* 统计卡片 */}
      <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
        <StatCard
          label={t('total_materials')}
          value={stats.totalMaterials}
          icon={<Package className="w-6 h-6" />}
          variant="default"
        />
        <StatCard
          label={t('total_quantity')}
          value={stats.totalQuantity}
          icon={<Layers className="w-6 h-6" />}
          variant="default"
        />
        <StatCard
          label={t('expiring_soon')}
          value={stats.expiringCount}
          icon={<AlertTriangle className="w-6 h-6" />}
          variant="warning"
        />
        <StatCard
          label={t('expired')}
          value={stats.expiredCount}
          icon={<XCircle className="w-6 h-6" />}
          variant="error"
        />
      </div>

      {/* 图表区域 */}
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
        {/* 柱状图 */}
        <Card className="lg:col-span-2 bg-surface-container-low border-outline-variant">
          <CardHeader className="pb-2">
            <CardTitle className="text-base font-medium text-on-surface">
              {t('by_category')}
            </CardTitle>
          </CardHeader>
          <CardContent>
            {stats.byCategory.length === 0 ? (
              <div className="flex h-40 items-center justify-center text-on-surface-variant text-sm">
                {t('no_data')}
              </div>
            ) : (
              <ReactECharts
                option={barOption}
                style={{ height: `${barHeight}px` }}
                notMerge
              />
            )}
          </CardContent>
        </Card>

        {/* 饼图 */}
        <Card className="bg-surface-container-low border-outline-variant">
          <CardHeader className="pb-2">
            <CardTitle className="text-base font-medium text-on-surface">
              {t('status_distribution')}
            </CardTitle>
          </CardHeader>
          <CardContent>
            {stats.totalMaterials === 0 ? (
              <div className="flex h-40 items-center justify-center text-on-surface-variant text-sm">
                {t('no_data')}
              </div>
            ) : (
              <ReactECharts
                option={pieOption}
                style={{ height: '280px' }}
                notMerge
              />
            )}
          </CardContent>
        </Card>
      </div>

      {/* 最近动态 */}
      <Card className="bg-surface-container-low border-outline-variant">
        <CardHeader className="pb-2">
          <CardTitle className="text-base font-medium text-on-surface">
            {t('recent_movements')}
          </CardTitle>
        </CardHeader>
        <CardContent>
          {stats.recentMovements.length === 0 ? (
            <div className="flex h-24 items-center justify-center text-on-surface-variant text-sm">
              {t('no_data')}
            </div>
          ) : (
            <div className="space-y-2">
              {stats.recentMovements.map((mv) => (
                <div
                  key={mv.id}
                  className="flex items-center justify-between gap-4 rounded-lg px-3 py-2 bg-surface-container hover:bg-surface-container-high transition-colors"
                >
                  <div className="flex items-center gap-3 min-w-0">
                    <Badge variant={movementBadgeVariant(mv.movement_type)} className="shrink-0">
                      {movementTypeLabel(mv.movement_type)}
                    </Badge>
                    <span className="text-sm text-on-surface truncate">
                      {mv.material?.name ?? mv.material_id}
                    </span>
                    {mv.reason ? (
                      <span className="text-xs text-on-surface-variant truncate hidden sm:block">
                        {mv.reason}
                      </span>
                    ) : null}
                  </div>
                  <div className="flex items-center gap-3 shrink-0 text-right">
                    <span
                      className={cn(
                        'text-sm font-medium tabular-nums',
                        mv.quantity_delta > 0 ? 'text-primary' : 'text-error',
                      )}
                    >
                      {mv.quantity_delta > 0 ? '+' : ''}{mv.quantity_delta} {mv.unit}
                    </span>
                    <span className="text-xs text-on-surface-variant hidden md:block">
                      {formatDateTime(mv.created_at)}
                    </span>
                  </div>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
