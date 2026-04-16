import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { notificationsApi } from '@/api/notifications'
import type { NotificationRecord } from '@/types/notification'

const STATUS_FILTER_OPTIONS = ['all', 'sent', 'failed'] as const

function StatusBadge({ status }: { status: NotificationRecord['status'] }) {
  const { t } = useTranslation('settings')
  if (status === 'sent')
    return <Badge variant="default">{t('notifStatusSent')}</Badge>
  if (status === 'failed')
    return <Badge variant="destructive">{t('notifStatusFailed')}</Badge>
  return <Badge variant="secondary">{t('notifStatusPending')}</Badge>
}

function materialLabel(record: NotificationRecord): string {
  const m = record.lot?.material
  if (!m) return record.lot_id
  return m.spec ? `${m.name}(${m.spec})` : m.name
}

export function NotificationHistorySection() {
  const { t, i18n } = useTranslation('settings')
  const [items, setItems] = useState<NotificationRecord[]>([])
  const [total, setTotal] = useState(0)
  const [statusFilter, setStatusFilter] = useState<string>('all')
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    setLoading(true)
    notificationsApi
      .list({ status: statusFilter === 'all' ? undefined : statusFilter, page: 1, page_size: 20 })
      .then((page) => {
        setItems(page.items)
        setTotal(page.total)
      })
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [statusFilter])

  return (
    <Card className="rounded-xl border-outline-variant/20 bg-surface-container-lowest shadow-sm">
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between gap-3 flex-wrap">
          <div>
            <CardTitle className="text-lg font-bold tracking-tight">{t('notifHistoryTitle')}</CardTitle>
            <CardDescription>{t('notifHistoryDescription')}</CardDescription>
          </div>
          <Select value={statusFilter} onValueChange={setStatusFilter}>
            <SelectTrigger className="w-32 h-8 text-sm">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {STATUS_FILTER_OPTIONS.map((v) => (
                <SelectItem key={v} value={v}>
                  {v === 'all' ? t('notifFilterAll') : v === 'sent' ? t('notifFilterSent') : t('notifFilterFailed')}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </CardHeader>
      <CardContent className="p-0">
        {loading ? (
          <div className="px-4 py-6 text-sm text-on-surface-variant text-center">
            加载中…
          </div>
        ) : items.length === 0 ? (
          <div className="px-4 py-6 text-sm text-on-surface-variant text-center">
            {t('notifHistoryEmpty')}
          </div>
        ) : (
          <>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t('notifColTime')}</TableHead>
                  <TableHead>{t('notifColMaterial')}</TableHead>
                  <TableHead>{t('notifColStatus')}</TableHead>
                  <TableHead className="hidden sm:table-cell">{t('notifColChannel')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {items.map((record) => (
                  <TableRow key={record.id}>
                    <TableCell className="text-sm whitespace-nowrap">
                      {new Date(record.notify_at).toLocaleString(i18n.language)}
                    </TableCell>
                    <TableCell className="text-sm max-w-[180px] truncate">
                      {materialLabel(record)}
                    </TableCell>
                    <TableCell>
                      <StatusBadge status={record.status} />
                    </TableCell>
                    <TableCell className="hidden sm:table-cell text-sm text-on-surface-variant">
                      {record.channel}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
            {total > 20 && (
              <div className="px-4 py-2 text-xs text-on-surface-variant">
                共 {total} 条，当前显示最新 20 条
              </div>
            )}
          </>
        )}
      </CardContent>
    </Card>
  )
}
