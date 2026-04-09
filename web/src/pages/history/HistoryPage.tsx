import { useCallback, useEffect, useState } from 'react'
import { RefreshCw, ChevronLeft, ChevronRight, Info } from 'lucide-react'
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
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { cn } from '@/lib/utils'
import { formatDateTime } from '@/lib/format'
import { useDebounce } from '@/hooks/useDebounce'
import { auditLogApi } from '@/api/auditLog'
import type { AuditLog, AuditAction } from '@/types/audit'
import { parseChangesDetail } from '@/types/audit'

// ── Constants ─────────────────────────────────────────────────────────────────

const PAGE_SIZE = 20

// ── Helpers ───────────────────────────────────────────────────────────────────

function actionLabel(action: AuditAction): string {
  switch (action) {
    case 'create': return '新增'
    case 'update': return '更新'
    case 'delete': return '删除'
  }
}

function actionVariant(action: AuditAction): 'default' | 'secondary' | 'destructive' {
  switch (action) {
    case 'create': return 'default'
    case 'update': return 'secondary'
    case 'delete': return 'destructive'
  }
}

function entityTypeLabel(type: string): string {
  return type === 'item' ? '物料' : '分类'
}

function channelLabel(channel: string): string {
  return channel === 'feishu' ? '飞书' : 'Web'
}

function changesSummary(log: AuditLog): string {
  const detail = parseChangesDetail(log.changes_detail)
  if (log.action === 'create' && detail.after) {
    const after = detail.after as Record<string, unknown>
    const parts: string[] = []
    if (after.quantity !== undefined) parts.push(`数量: ${after.quantity}${after.unit ?? ''}`)
    if (after.location) parts.push(`位置: ${after.location}`)
    return parts.join('，') || '—'
  }
  if (log.action === 'delete' && detail.before) {
    const before = detail.before as Record<string, unknown>
    const parts: string[] = []
    if (before.quantity !== undefined) parts.push(`数量: ${before.quantity}${before.unit ?? ''}`)
    if (before.location) parts.push(`位置: ${before.location}`)
    return parts.join('，') || '—'
  }
  if (log.action === 'update' && detail.before && detail.after) {
    const before = detail.before as Record<string, unknown>
    const after = detail.after as Record<string, unknown>
    const diffs: string[] = []
    const fields: Array<[string, string]> = [
      ['quantity', '数量'],
      ['unit', '单位'],
      ['location', '位置'],
      ['name', '名称'],
      ['expire_at', '过期日'],
      ['notes', '备注'],
    ]
    for (const [key, label] of fields) {
      const bVal = before[key]
      const aVal = after[key]
      if (bVal !== aVal) {
        diffs.push(`${label}: ${bVal ?? '—'} → ${aVal ?? '—'}`)
      }
      if (diffs.length >= 2) break
    }
    return diffs.join('，') || '—'
  }
  return '—'
}

// ── Pagination ────────────────────────────────────────────────────────────────

function TablePagination({
  page,
  totalPages,
  totalItems,
  pageSize,
  onChange,
}: {
  page: number
  totalPages: number
  totalItems: number
  pageSize: number
  onChange: (p: number) => void
}) {
  const start = Math.min((page - 1) * pageSize + 1, totalItems)
  const end = Math.min(page * pageSize, totalItems)

  const pages: (number | '…')[] = []
  if (totalPages <= 7) {
    for (let i = 1; i <= totalPages; i++) pages.push(i)
  } else {
    pages.push(1)
    if (page > 3) pages.push('…')
    for (let i = Math.max(2, page - 1); i <= Math.min(totalPages - 1, page + 1); i++) pages.push(i)
    if (page < totalPages - 2) pages.push('…')
    pages.push(totalPages)
  }

  return (
    <div className="px-6 py-4 flex items-center justify-between border-t border-outline-variant/20">
      <p className="text-xs text-on-surface-variant">
        显示 <span className="font-medium text-on-surface">{start} – {end}</span> / 共 {totalItems} 条
      </p>
      <div className="flex items-center gap-1">
        <button
          onClick={() => onChange(page - 1)}
          disabled={page === 1}
          className="w-8 h-8 flex items-center justify-center rounded-lg text-on-surface-variant hover:bg-surface-container transition-colors disabled:opacity-30 disabled:cursor-not-allowed cursor-pointer"
        >
          <ChevronLeft className="w-4 h-4" />
        </button>
        {pages.map((p, i) =>
          p === '…' ? (
            <span key={`ellipsis-${i}`} className="w-8 h-8 flex items-center justify-center text-outline-variant text-xs">…</span>
          ) : (
            <button
              key={p}
              onClick={() => onChange(p)}
              className={cn(
                'w-8 h-8 flex items-center justify-center rounded-lg text-xs font-bold transition-colors cursor-pointer',
                p === page
                  ? 'bg-primary text-on-primary'
                  : 'text-on-surface-variant hover:bg-surface-container'
              )}
            >
              {p}
            </button>
          )
        )}
        <button
          onClick={() => onChange(page + 1)}
          disabled={page === totalPages}
          className="w-8 h-8 flex items-center justify-center rounded-lg text-on-surface-variant hover:bg-surface-container transition-colors disabled:opacity-30 disabled:cursor-not-allowed cursor-pointer"
        >
          <ChevronRight className="w-4 h-4" />
        </button>
      </div>
    </div>
  )
}

// ── Main page ──────────────────────────────────────────────────────────────────

export default function HistoryPage() {
  const [logs, setLogs] = useState<AuditLog[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Filters
  const [filterAction, setFilterAction] = useState('__all__')
  const [filterChannel, setFilterChannel] = useState('__all__')
  const [filterUserName, setFilterUserName] = useState('')
  const [startDate, setStartDate] = useState('')
  const [endDate, setEndDate] = useState('')

  // Detail dialog
  const [detailLog, setDetailLog] = useState<AuditLog | null>(null)

  const debouncedUserName = useDebounce(filterUserName, 300)

  const loadData = useCallback(async (currentPage: number) => {
    setLoading(true)
    setError(null)
    try {
      const result = await auditLogApi.list({
        action: filterAction === '__all__' ? undefined : filterAction,
        channel: filterChannel === '__all__' ? undefined : filterChannel,
        user_name: debouncedUserName || undefined,
        start_date: startDate || undefined,
        end_date: endDate || undefined,
        page: currentPage,
        page_size: PAGE_SIZE,
      })
      setLogs(result.logs ?? [])
      setTotal(result.total)
    } catch (e) {
      setError(e instanceof Error ? e.message : '加载失败')
    } finally {
      setLoading(false)
    }
  }, [filterAction, filterChannel, debouncedUserName, startDate, endDate])

  // Reset to page 1 when filters change
  useEffect(() => {
    setPage(1)
  }, [filterAction, filterChannel, debouncedUserName, startDate, endDate])

  useEffect(() => {
    loadData(page)
  }, [loadData, page])

  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-on-surface">使用记录</h1>
          <p className="text-sm text-on-surface-variant mt-0.5">所有库存操作的历史审计日志</p>
        </div>
        <Button variant="outline" size="sm" onClick={() => loadData(page)} disabled={loading}>
          <RefreshCw className={cn('w-4 h-4 mr-2', loading && 'animate-spin')} />
          刷新
        </Button>
      </div>

      {/* Filters */}
      <div className="flex flex-wrap gap-3">
        <Select value={filterAction} onValueChange={setFilterAction}>
          <SelectTrigger className="w-36">
            <SelectValue placeholder="全部操作" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="__all__">全部操作</SelectItem>
            <SelectItem value="create">新增</SelectItem>
            <SelectItem value="update">更新</SelectItem>
            <SelectItem value="delete">删除</SelectItem>
          </SelectContent>
        </Select>

        <Select value={filterChannel} onValueChange={setFilterChannel}>
          <SelectTrigger className="w-36">
            <SelectValue placeholder="全部渠道" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="__all__">全部渠道</SelectItem>
            <SelectItem value="web">Web</SelectItem>
            <SelectItem value="feishu">飞书</SelectItem>
          </SelectContent>
        </Select>

        <Input
          placeholder="搜索操作者…"
          value={filterUserName}
          onChange={(e) => setFilterUserName(e.target.value)}
          className="w-44"
        />

        <Input
          type="date"
          value={startDate}
          onChange={(e) => setStartDate(e.target.value)}
          className="w-40"
          title="开始日期"
        />
        <Input
          type="date"
          value={endDate}
          onChange={(e) => setEndDate(e.target.value)}
          className="w-40"
          title="结束日期"
        />
      </div>

      {/* Error */}
      {error && (
        <div className="rounded-lg bg-error-container px-4 py-3 text-sm text-error">
          {error}
        </div>
      )}

      {/* Table */}
      <div className="rounded-xl border border-outline-variant/20 bg-surface-container-lowest shadow-sm overflow-hidden">
        <Table>
          <TableHeader>
            <TableRow className="border-outline-variant/20">
              <TableHead className="text-xs font-semibold text-on-surface-variant">时间</TableHead>
              <TableHead className="text-xs font-semibold text-on-surface-variant">操作者</TableHead>
              <TableHead className="text-xs font-semibold text-on-surface-variant">渠道</TableHead>
              <TableHead className="text-xs font-semibold text-on-surface-variant">操作</TableHead>
              <TableHead className="text-xs font-semibold text-on-surface-variant">类型</TableHead>
              <TableHead className="text-xs font-semibold text-on-surface-variant">名称</TableHead>
              <TableHead className="text-xs font-semibold text-on-surface-variant">变更摘要</TableHead>
              <TableHead className="text-xs font-semibold text-on-surface-variant w-16">详情</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {logs.length === 0 ? (
              <TableRow>
                <TableCell colSpan={8} className="text-center py-12 text-sm text-on-surface-variant">
                  {loading ? '加载中…' : '暂无记录'}
                </TableCell>
              </TableRow>
            ) : (
              logs.map((log) => (
                <TableRow key={log.id} className="border-outline-variant/10 hover:bg-surface-container/50">
                  <TableCell className="text-xs text-on-surface-variant whitespace-nowrap">
                    {formatDateTime(log.created_at)}
                  </TableCell>
                  <TableCell className="text-xs text-on-surface">
                    {log.user_name || log.user_id || '—'}
                  </TableCell>
                  <TableCell>
                    <Badge variant={log.channel === 'feishu' ? 'secondary' : 'outline'} className="text-xs">
                      {channelLabel(log.channel)}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <Badge variant={actionVariant(log.action as AuditAction)} className="text-xs">
                      {actionLabel(log.action as AuditAction)}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-xs text-on-surface-variant">
                    {entityTypeLabel(log.entity_type)}
                  </TableCell>
                  <TableCell className="text-xs font-medium text-on-surface max-w-35 truncate" title={log.entity_name}>
                    {log.entity_name || '—'}
                  </TableCell>
                  <TableCell className="text-xs text-on-surface-variant max-w-50 truncate" title={changesSummary(log)}>
                    {changesSummary(log)}
                  </TableCell>
                  <TableCell>
                    <Button
                      variant="ghost"
                      size="icon"
                      className="w-7 h-7"
                      onClick={() => setDetailLog(log)}
                    >
                      <Info className="w-3.5 h-3.5" />
                    </Button>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>

        {total > 0 && (
          <TablePagination
            page={page}
            totalPages={totalPages}
            totalItems={total}
            pageSize={PAGE_SIZE}
            onChange={setPage}
          />
        )}
      </div>

      {/* Detail dialog */}
      <Dialog open={detailLog !== null} onOpenChange={(open) => { if (!open) setDetailLog(null) }}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>变更详情</DialogTitle>
          </DialogHeader>
          {detailLog && (
            <div className="space-y-3 text-sm">
              <div className="grid grid-cols-2 gap-2 text-xs">
                <span className="text-on-surface-variant">时间</span>
                <span className="text-on-surface">{formatDateTime(detailLog.created_at)}</span>
                <span className="text-on-surface-variant">操作者</span>
                <span className="text-on-surface">{detailLog.user_name || detailLog.user_id || '—'}</span>
                <span className="text-on-surface-variant">渠道</span>
                <span className="text-on-surface">{channelLabel(detailLog.channel)}</span>
                <span className="text-on-surface-variant">操作</span>
                <span className="text-on-surface">{actionLabel(detailLog.action as AuditAction)}</span>
                <span className="text-on-surface-variant">对象</span>
                <span className="text-on-surface">{entityTypeLabel(detailLog.entity_type)}：{detailLog.entity_name}</span>
              </div>
              {detailLog.changes_detail && (
                <>
                  <p className="text-xs font-semibold text-on-surface-variant pt-1">完整变更数据</p>
                  <pre className="text-xs bg-surface-container rounded-lg p-3 overflow-auto max-h-64 text-on-surface whitespace-pre-wrap break-all">
                    {JSON.stringify(parseChangesDetail(detailLog.changes_detail), null, 2)}
                  </pre>
                </>
              )}
            </div>
          )}
        </DialogContent>
      </Dialog>
    </div>
  )
}
