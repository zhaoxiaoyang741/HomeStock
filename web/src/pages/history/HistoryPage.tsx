import { useCallback, useEffect, useMemo, useState } from 'react'
import { Info, RefreshCw } from 'lucide-react'
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

const REASON_MAP: Record<string, string> = {
  inbound: '入库',
  consume: '消耗',
  'manual consume': '消耗',
  adjustment: '调整',
  void: '作废',
}

function reasonLabel(reason: string, remark: string): string {
  const r = reason || remark
  return REASON_MAP[r.toLowerCase()] ?? (r || '—')
}

function channelLabel(channel: string): string {
  switch (channel) {
    case 'web': return '网页'
    case 'feishu': return '飞书'
    case 'system': return '系统'
    default: return channel || '系统'
  }
}

function OperatorCell({ userName, userId, channel }: { userName: string; userId: string; channel: string }) {
  const name = userName || userId || ''
  const label = channelLabel(channel)
  return (
    <div className="flex flex-col gap-0.5">
      <span className="text-sm text-on-surface">{name || label}</span>
      {name && (
        <span className="text-xs text-on-surface-variant">{label}</span>
      )}
    </div>
  )
}

function movementLabel(type: StockMovementType): string {
  switch (type) {
    case 'inbound':
      return '入库'
    case 'consume':
      return '消耗'
    case 'adjustment':
      return '调整'
    case 'void':
      return '作废'
  }
}

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

function actionLabel(action: AuditAction): string {
  switch (action) {
    case 'create':
      return '新增'
    case 'update':
      return '更新'
    case 'delete':
      return '删除'
  }
}

function auditEntityLabel(type: string): string {
  switch (type) {
    case 'material':
      return '物料'
    case 'stock_lot':
      return '批次'
    case 'category':
      return '分类'
    default:
      return type
  }
}

export default function HistoryPage() {
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
      setError(err instanceof Error ? err.message : '加载历史失败')
    } finally {
      setLoading(false)
    }
  }, [movementType, auditAction, auditChannel, startDate, endDate])

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
          <h1 className="text-2xl font-extrabold tracking-tight text-on-surface">库存历史</h1>
          <p className="text-sm text-on-surface-variant mt-0.5">库存流水作为主历史，审计日志保留为后台操作追踪。</p>
        </div>
        <Button variant="outline" size="sm" onClick={() => void loadData()} disabled={loading}>
          <RefreshCw className={cn('w-4 h-4 mr-2', loading && 'animate-spin')} />
          刷新
        </Button>
      </div>

      <div className="flex flex-wrap gap-3">
        <Input
          className="w-56"
          value={search}
          onChange={(event) => setSearch(event.target.value)}
          placeholder="搜索物料、批次或操作人"
        />
        <Select value={movementType} onValueChange={setMovementType}>
          <SelectTrigger className="w-36">
            <SelectValue placeholder="全部流水类型" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="__all__">全部流水类型</SelectItem>
            <SelectItem value="inbound">入库</SelectItem>
            <SelectItem value="consume">消耗</SelectItem>
            <SelectItem value="adjustment">调整</SelectItem>
          </SelectContent>
        </Select>
        <Select value={auditAction} onValueChange={setAuditAction}>
          <SelectTrigger className="w-32">
            <SelectValue placeholder="全部审计动作" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="__all__">全部动作</SelectItem>
            <SelectItem value="create">新增</SelectItem>
            <SelectItem value="update">更新</SelectItem>
            <SelectItem value="delete">删除</SelectItem>
          </SelectContent>
        </Select>
        <Select value={auditChannel} onValueChange={setAuditChannel}>
          <SelectTrigger className="w-32">
            <SelectValue placeholder="全部渠道" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="__all__">全部渠道</SelectItem>
            <SelectItem value="web">Web</SelectItem>
            <SelectItem value="feishu">飞书</SelectItem>
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
          <TabsTrigger value="movements">库存流水</TabsTrigger>
          <TabsTrigger value="audit">操作审计</TabsTrigger>
        </TabsList>

        <TabsContent value="movements">
          <div className="rounded-xl border border-outline-variant/20 bg-surface-container-lowest shadow-sm overflow-hidden">
            <Table>
              <TableHeader>
                <TableRow className="border-outline-variant/20">
                  <TableHead className="text-xs font-bold uppercase tracking-widest text-on-surface-variant">时间</TableHead>
                  <TableHead className="text-xs font-bold uppercase tracking-widest text-on-surface-variant">类型</TableHead>
                  <TableHead className="text-xs font-bold uppercase tracking-widest text-on-surface-variant">物料</TableHead>
                  <TableHead className="text-xs font-bold uppercase tracking-widest text-on-surface-variant">批次</TableHead>
                  <TableHead className="text-xs font-bold uppercase tracking-widest text-on-surface-variant">变动</TableHead>
                  <TableHead className="text-xs font-bold uppercase tracking-widest text-on-surface-variant">原因</TableHead>
                  <TableHead className="text-xs font-bold uppercase tracking-widest text-on-surface-variant">操作人</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {filteredMovements.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={7} className="h-32 text-center text-on-surface-variant">
                      {loading ? '加载中…' : '暂无库存流水'}
                    </TableCell>
                  </TableRow>
                ) : (
                  filteredMovements.map((movement) => (
                    <TableRow key={movement.id} className="border-outline-variant/10 hover:bg-surface">
                      <TableCell className="text-xs text-on-surface-variant whitespace-nowrap">
                        {formatDateTime(movement.created_at)}
                      </TableCell>
                      <TableCell>
                        <Badge variant={movementVariant(movement.movement_type)}>{movementLabel(movement.movement_type)}</Badge>
                      </TableCell>
                      <TableCell className="text-sm text-on-surface">
                        <div className="font-medium">{movement.material?.name ?? '—'}</div>
                        <div className="text-xs text-on-surface-variant">{movement.material?.spec || '未填写规格'}</div>
                      </TableCell>
                      <TableCell className="text-xs text-on-surface-variant">{movement.lot_id}</TableCell>
                      <TableCell className={cn('font-semibold', movement.quantity_delta < 0 ? 'text-error' : 'text-primary')}>
                        {movement.quantity_delta > 0 ? '+' : ''}{movement.quantity_delta} {movement.unit}
                      </TableCell>
                      <TableCell className="text-sm text-on-surface-variant">{reasonLabel(movement.reason, movement.remark)}</TableCell>
                      <TableCell><OperatorCell userName={movement.user_name} userId={movement.user_id} channel={movement.channel} /></TableCell>
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
                  <TableHead className="text-xs font-bold uppercase tracking-widest text-on-surface-variant">时间</TableHead>
                  <TableHead className="text-xs font-bold uppercase tracking-widest text-on-surface-variant">操作人</TableHead>
                  <TableHead className="text-xs font-bold uppercase tracking-widest text-on-surface-variant">动作</TableHead>
                  <TableHead className="text-xs font-bold uppercase tracking-widest text-on-surface-variant">对象</TableHead>
                  <TableHead className="text-xs font-bold uppercase tracking-widest text-on-surface-variant">名称</TableHead>
                  <TableHead className="text-xs font-bold uppercase tracking-widest text-on-surface-variant text-right">详情</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {filteredAuditLogs.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={6} className="h-32 text-center text-on-surface-variant">
                      {loading ? '加载中…' : '暂无审计日志'}
                    </TableCell>
                  </TableRow>
                ) : (
                  filteredAuditLogs.map((log) => (
                    <TableRow key={log.id} className="border-outline-variant/10 hover:bg-surface">
                      <TableCell className="text-xs text-on-surface-variant whitespace-nowrap">
                        {formatDateTime(log.created_at)}
                      </TableCell>
                      <TableCell><OperatorCell userName={log.user_name} userId={log.user_id} channel={log.channel} /></TableCell>
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
            <DialogTitle>审计日志详情</DialogTitle>
          </DialogHeader>
          {selectedAuditLog && (
            <div className="space-y-3 text-sm">
              <div className="grid grid-cols-2 gap-2 text-xs">
                <span className="text-on-surface-variant">时间</span>
                <span>{formatDateTime(selectedAuditLog.created_at)}</span>
                <span className="text-on-surface-variant">操作人</span>
                <span>{selectedAuditLog.user_name || selectedAuditLog.user_id || '系统'}</span>
                <span className="text-on-surface-variant">动作</span>
                <span>{actionLabel(selectedAuditLog.action)}</span>
                <span className="text-on-surface-variant">对象</span>
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
