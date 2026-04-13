import { useCallback, useEffect, useMemo, useState } from 'react'
import { Info, RefreshCw } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { cn } from '@/lib/utils'
import { formatDateTime } from '@/lib/format'
import { useDebounce } from '@/hooks/useDebounce'
import { stockMovementApi } from '@/api/stockMovement'
import { auditLogApi } from '@/api/auditLog'
import type { StockMovement, StockMovementType } from '@/types/stock'
import type { AuditAction, AuditLog } from '@/types/audit'
import { parseChangesDetail } from '@/types/audit'

function movementVariant(type: StockMovementType): 'default' | 'secondary' | 'destructive' {
  switch (type) {
    case 'inbound':
      return 'default'
    case 'consume':
      return 'secondary'
    case 'adjustment':
    case 'void':
      return 'destructive'
  }
}

function OperatorCell({ userName, userId, channelDisplayLabel }: { userName: string; userId: string; channelDisplayLabel: string }) {
  const name = userName || userId || ''
  return (
    <div className="flex flex-col gap-0.5">
      <span className="text-sm text-on-surface">{name || channelDisplayLabel}</span>
      {name && (
        <span className="text-xs text-on-surface-variant">{channelDisplayLabel}</span>
      )}
    </div>
  )
}

export default function HistoryPage() {
  const { t, i18n } = useTranslation('history')
  const [movements, setMovements] = useState<StockMovement[]>([])
  const [auditLogs, setAuditLogs] = useState<AuditLog[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [movementType, setMovementType] = useState('__all__')
  const [auditAction, setAuditAction] = useState('__all__')
  const [auditChannel, setAuditChannel] = useState('__all__')
  const [search, setSearch] = useState('')
  const [startDate, setStartDate] = useState('')
  const [endDate, setEndDate] = useState('')
  const [selectedAuditLog, setSelectedAuditLog] = useState<AuditLog | null>(null)
  const debouncedSearch = useDebounce(search, 300)

  const REASON_MAP: Record<string, string> = {
    inbound: t('reasonInbound'),
    consume: t('reasonConsume'),
    'manual consume': t('reasonConsume'),
    adjustment: t('reasonAdjustment'),
    void: t('reasonVoid'),
  }

  function reasonLabel(reason: string, remark: string): string {
    const r = reason || remark
    return REASON_MAP[r.toLowerCase()] ?? (r || '—')
  }

  function channelLabel(channel: string): string {
    switch (channel) {
      case 'web': return t('channelWeb')
      case 'feishu': return t('channelFeishu')
      case 'system': return t('channelSystem')
      default: return channel || t('channelSystem')
    }
  }

  function movementLabel(type: StockMovementType): string {
    switch (type) {
      case 'inbound': return t('movementInbound')
      case 'consume': return t('movementConsume')
      case 'adjustment': return t('movementAdjustment')
      case 'void': return t('movementVoid')
    }
  }

  function actionLabel(action: AuditAction): string {
    switch (action) {
      case 'create': return t('actionCreate')
      case 'update': return t('actionUpdate')
      case 'delete': return t('actionDelete')
    }
  }

  function auditEntityLabel(type: string): string {
    switch (type) {
      case 'material': return t('entityMaterial')
      case 'stock_lot': return t('entityStockLot')
      case 'category': return t('entityCategory')
      default: return type
    }
  }

  const loadData = useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      const [movementRes, auditRes] = await Promise.all([
        stockMovementApi.list({
          movement_type: movementType !== '__all__' ? movementType : undefined,
          start_date: startDate || undefined,
          end_date: endDate || undefined,
        }),
        auditLogApi.list({
          action: auditAction !== '__all__' ? auditAction : undefined,
          channel: auditChannel !== '__all__' ? auditChannel : undefined,
          start_date: startDate || undefined,
          end_date: endDate || undefined,
          page: 1,
          page_size: 100,
        }),
      ])
      setMovements(movementRes.movements)
      setAuditLogs(auditRes.logs)
    } catch (err) {
      setError(err instanceof Error ? err.message : t('loadingFailed'))
    } finally {
      setLoading(false)
    }
  }, [movementType, auditAction, auditChannel, startDate, endDate, t])

  useEffect(() => {
    void loadData()
  }, [loadData])

  const filteredMovements = useMemo(() => {
    const query = debouncedSearch.trim().toLowerCase()
    if (!query) return movements
    return movements.filter((movement) => {
      const materialName = movement.material?.name?.toLowerCase() ?? ''
      const materialSpec = movement.material?.spec?.toLowerCase() ?? ''
      const lotID = movement.lot_id.toLowerCase()
      const reason = movement.reason.toLowerCase()
      const remark = movement.remark.toLowerCase()
      return [materialName, materialSpec, lotID, reason, remark].some((field) => field.includes(query))
    })
  }, [movements, debouncedSearch])

  const filteredAuditLogs = useMemo(() => {
    const query = debouncedSearch.trim().toLowerCase()
    if (!query) return auditLogs
    return auditLogs.filter((log) =>
      [log.entity_name, log.user_name, log.user_id, log.entity_type]
        .filter(Boolean)
        .some((field) => field.toLowerCase().includes(query)),
    )
  }, [auditLogs, debouncedSearch])

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between gap-4 flex-wrap">
        <div>
          <h1 className="text-2xl font-extrabold tracking-tight text-on-surface">{t('title')}</h1>
          <p className="text-sm text-on-surface-variant mt-0.5">{t('subtitle')}</p>
        </div>
        <Button variant="outline" size="sm" onClick={() => void loadData()} disabled={loading}>
          <RefreshCw className={cn('w-4 h-4 mr-2', loading && 'animate-spin')} />
          {t('common:refresh')}
        </Button>
      </div>

      <div className="flex flex-wrap gap-3">
        <Input
          className="w-56"
          value={search}
          onChange={(event) => setSearch(event.target.value)}
          placeholder={t('searchPlaceholder')}
        />
        <Select value={movementType} onValueChange={setMovementType}>
          <SelectTrigger className="w-36">
            <SelectValue placeholder={t('allMovementTypes')} />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="__all__">{t('allMovementTypes')}</SelectItem>
            <SelectItem value="inbound">{t('movementInbound')}</SelectItem>
            <SelectItem value="consume">{t('movementConsume')}</SelectItem>
            <SelectItem value="adjustment">{t('movementAdjustment')}</SelectItem>
          </SelectContent>
        </Select>
        <Select value={auditAction} onValueChange={setAuditAction}>
          <SelectTrigger className="w-32">
            <SelectValue placeholder={t('allAuditActions')} />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="__all__">{t('allAuditActions')}</SelectItem>
            <SelectItem value="create">{t('actionCreate')}</SelectItem>
            <SelectItem value="update">{t('actionUpdate')}</SelectItem>
            <SelectItem value="delete">{t('actionDelete')}</SelectItem>
          </SelectContent>
        </Select>
        <Select value={auditChannel} onValueChange={setAuditChannel}>
          <SelectTrigger className="w-32">
            <SelectValue placeholder={t('allChannels')} />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="__all__">{t('allChannels')}</SelectItem>
            <SelectItem value="web">{t('channelWeb')}</SelectItem>
            <SelectItem value="feishu">{t('channelFeishu')}</SelectItem>
          </SelectContent>
        </Select>
        <Input type="date" value={startDate} onChange={(event) => setStartDate(event.target.value)} className="w-40" />
        <Input type="date" value={endDate} onChange={(event) => setEndDate(event.target.value)} className="w-40" />
      </div>

      {error && (
        <div className="rounded-lg bg-error-container px-4 py-3 text-sm text-error">{error}</div>
      )}

      <Tabs defaultValue="movements" className="space-y-3">
        <TabsList className="bg-surface-container">
          <TabsTrigger value="movements">{t('tabMovements')}</TabsTrigger>
          <TabsTrigger value="audit">{t('tabAudit')}</TabsTrigger>
        </TabsList>

        <TabsContent value="movements">
          <div className="rounded-xl border border-outline-variant/20 bg-surface-container-lowest shadow-sm overflow-hidden">
            <Table>
              <TableHeader>
                <TableRow className="border-outline-variant/20">
                  <TableHead className="text-xs font-bold uppercase tracking-widest text-on-surface-variant">{t('colTime')}</TableHead>
                  <TableHead className="text-xs font-bold uppercase tracking-widest text-on-surface-variant">{t('colType')}</TableHead>
                  <TableHead className="text-xs font-bold uppercase tracking-widest text-on-surface-variant">{t('colMaterial')}</TableHead>
                  <TableHead className="text-xs font-bold uppercase tracking-widest text-on-surface-variant">{t('colLot')}</TableHead>
                  <TableHead className="text-xs font-bold uppercase tracking-widest text-on-surface-variant">{t('colDelta')}</TableHead>
                  <TableHead className="text-xs font-bold uppercase tracking-widest text-on-surface-variant">{t('colReason')}</TableHead>
                  <TableHead className="text-xs font-bold uppercase tracking-widest text-on-surface-variant">{t('colOperator')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {filteredMovements.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={7} className="h-32 text-center text-on-surface-variant">
                      {loading ? t('common:loading') : t('noMovements')}
                    </TableCell>
                  </TableRow>
                ) : (
                  filteredMovements.map((movement) => (
                    <TableRow key={movement.id} className="border-outline-variant/10 hover:bg-surface">
                      <TableCell className="text-xs text-on-surface-variant whitespace-nowrap">
                        {formatDateTime(movement.created_at, i18n.language)}
                      </TableCell>
                      <TableCell>
                        <Badge variant={movementVariant(movement.movement_type)}>{movementLabel(movement.movement_type)}</Badge>
                      </TableCell>
                      <TableCell className="text-sm text-on-surface">
                        <div className="font-medium">{movement.material?.name ?? '—'}</div>
                        <div className="text-xs text-on-surface-variant">{movement.material?.spec || t('noSpec')}</div>
                      </TableCell>
                      <TableCell className="text-xs text-on-surface-variant">{movement.lot_id}</TableCell>
                      <TableCell className={cn('font-semibold', movement.quantity_delta < 0 ? 'text-error' : 'text-primary')}>
                        {movement.quantity_delta > 0 ? '+' : ''}{movement.quantity_delta} {movement.unit}
                      </TableCell>
                      <TableCell className="text-sm text-on-surface-variant">{reasonLabel(movement.reason, movement.remark)}</TableCell>
                      <TableCell>
                        <OperatorCell
                          userName={movement.user_name}
                          userId={movement.user_id}
                          channelDisplayLabel={channelLabel(movement.channel)}
                        />
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </div>
        </TabsContent>

        <TabsContent value="audit">
          <div className="rounded-xl border border-outline-variant/20 bg-surface-container-lowest shadow-sm overflow-hidden">
            <Table>
              <TableHeader>
                <TableRow className="border-outline-variant/20">
                  <TableHead className="text-xs font-bold uppercase tracking-widest text-on-surface-variant">{t('colTime')}</TableHead>
                  <TableHead className="text-xs font-bold uppercase tracking-widest text-on-surface-variant">{t('colOperator')}</TableHead>
                  <TableHead className="text-xs font-bold uppercase tracking-widest text-on-surface-variant">{t('colAction')}</TableHead>
                  <TableHead className="text-xs font-bold uppercase tracking-widest text-on-surface-variant">{t('colObject')}</TableHead>
                  <TableHead className="text-xs font-bold uppercase tracking-widest text-on-surface-variant">{t('colName')}</TableHead>
                  <TableHead className="text-xs font-bold uppercase tracking-widest text-on-surface-variant text-right">{t('colDetails')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {filteredAuditLogs.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={6} className="h-32 text-center text-on-surface-variant">
                      {loading ? t('common:loading') : t('noAuditLogs')}
                    </TableCell>
                  </TableRow>
                ) : (
                  filteredAuditLogs.map((log) => (
                    <TableRow key={log.id} className="border-outline-variant/10 hover:bg-surface">
                      <TableCell className="text-xs text-on-surface-variant whitespace-nowrap">
                        {formatDateTime(log.created_at, i18n.language)}
                      </TableCell>
                      <TableCell>
                        <OperatorCell
                          userName={log.user_name}
                          userId={log.user_id}
                          channelDisplayLabel={channelLabel(log.channel)}
                        />
                      </TableCell>
                      <TableCell>
                        <Badge variant={log.action === 'delete' ? 'destructive' : log.action === 'update' ? 'secondary' : 'default'}>
                          {actionLabel(log.action)}
                        </Badge>
                      </TableCell>
                      <TableCell className="text-sm text-on-surface-variant">{auditEntityLabel(log.entity_type)}</TableCell>
                      <TableCell className="text-sm text-on-surface">{log.entity_name || '—'}</TableCell>
                      <TableCell className="text-right">
                        <Button variant="ghost" size="icon" onClick={() => setSelectedAuditLog(log)}>
                          <Info className="w-4 h-4" />
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </div>
        </TabsContent>
      </Tabs>

      <Dialog open={selectedAuditLog !== null} onOpenChange={(nextOpen) => !nextOpen && setSelectedAuditLog(null)}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>{t('auditDialogTitle')}</DialogTitle>
          </DialogHeader>
          {selectedAuditLog && (
            <div className="space-y-3 text-sm">
              <div className="grid grid-cols-2 gap-2 text-xs">
                <span className="text-on-surface-variant">{t('auditFieldTime')}</span>
                <span>{formatDateTime(selectedAuditLog.created_at, i18n.language)}</span>
                <span className="text-on-surface-variant">{t('auditFieldOperator')}</span>
                <span>{selectedAuditLog.user_name || selectedAuditLog.user_id || t('channelSystem')}</span>
                <span className="text-on-surface-variant">{t('auditFieldAction')}</span>
                <span>{actionLabel(selectedAuditLog.action)}</span>
                <span className="text-on-surface-variant">{t('auditFieldObject')}</span>
                <span>{auditEntityLabel(selectedAuditLog.entity_type)} / {selectedAuditLog.entity_name || '—'}</span>
              </div>
              <pre className="text-xs bg-surface-container rounded-lg p-3 overflow-auto max-h-64 text-on-surface whitespace-pre-wrap break-all">
                {JSON.stringify(parseChangesDetail(selectedAuditLog.changes_detail), null, 2)}
              </pre>
            </div>
          )}
        </DialogContent>
      </Dialog>
    </div>
  )
}
