import { useCallback, useEffect, useState } from 'react'
import { Plus, Pencil, Trash2, Search, RefreshCw, Package, TriangleAlert, Tags, ChevronLeft, ChevronRight } from 'lucide-react'
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
import { cn } from '@/lib/utils'
import { formatDate } from '@/lib/format'
import { useDebounce } from '@/hooks/useDebounce'
import { itemApi } from '@/api/item'
import { categoryApi } from '@/api/category'
import type { Item, CreateItemPayload, UpdateItemPayload } from '@/types/item'
import type { Category } from '@/types/category'
import { getItemStatus, type ItemStatus } from '@/types/item'
import ItemFormDialog from './ItemFormDialog'
import DeleteConfirmDialog from './DeleteConfirmDialog'

// ── Constants ────────────────────────────────────────────────────────────────

const PAGE_SIZE = 10

// ── Status badge ──────────────────────────────────────────────────────────────

const STATUS_MAP: Record<ItemStatus, { label: string; variant: 'default' | 'outline' | 'secondary' | 'destructive' }> = {
  normal:   { label: '正常',   variant: 'default' },
  expiring: { label: '即将过期', variant: 'outline' },
  expired:  { label: '已过期',  variant: 'destructive' },
}

function StatusBadge({ status }: { status: ItemStatus }) {
  const { label, variant } = STATUS_MAP[status]
  return (
    <Badge
      variant={variant}
      className={cn(variant === 'outline' && 'text-tertiary border-tertiary/50')}
    >
      {label}
    </Badge>
  )
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
  const end   = Math.min(page * pageSize, totalItems)

  // Build page number array with ellipsis logic
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

// ── Summary cards ─────────────────────────────────────────────────────────────

function SummaryCards({ items, categories }: { items: Item[]; categories: Category[] }) {
  const warningCount = items.filter((i) => {
    const s = getItemStatus(i)
    return s === 'expiring' || s === 'expired'
  }).length

  return (
    <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
      <div className="p-5 bg-surface-container-lowest rounded-xl border border-outline-variant/20 flex items-center gap-4 shadow-sm">
        <div className="w-11 h-11 rounded-full bg-primary/10 flex items-center justify-center shrink-0">
          <Package className="w-5 h-5 text-primary" />
        </div>
        <div>
          <p className="text-[10px] font-bold uppercase tracking-widest text-on-surface-variant">总物料数</p>
          <p className="text-xl font-extrabold text-on-surface">{items.length} 件</p>
        </div>
      </div>

      <div className="p-5 bg-surface-container-lowest rounded-xl border border-outline-variant/20 flex items-center gap-4 shadow-sm">
        <div className="w-11 h-11 rounded-full bg-tertiary-container/20 flex items-center justify-center shrink-0">
          <TriangleAlert className="w-5 h-5 text-tertiary" />
        </div>
        <div>
          <p className="text-[10px] font-bold uppercase tracking-widest text-on-surface-variant">临期 / 过期预警</p>
          <p className={cn('text-xl font-extrabold', warningCount > 0 ? 'text-tertiary' : 'text-on-surface')}>
            {warningCount} 件物品
          </p>
        </div>
      </div>

      <div className="p-5 bg-surface-container-lowest rounded-xl border border-outline-variant/20 flex items-center gap-4 shadow-sm">
        <div className="w-11 h-11 rounded-full bg-secondary-container/20 flex items-center justify-center shrink-0">
          <Tags className="w-5 h-5 text-secondary" />
        </div>
        <div>
          <p className="text-[10px] font-bold uppercase tracking-widest text-on-surface-variant">分类数量</p>
          <p className="text-xl font-extrabold text-on-surface">{categories.length} 个</p>
        </div>
      </div>
    </div>
  )
}

// ── Main page ─────────────────────────────────────────────────────────────────

export default function InventoryPage() {
  const [items, setItems] = useState<Item[]>([])
  const [categories, setCategories] = useState<Category[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  // Filters
  const [search, setSearch] = useState('')
  const [filterCategory, setFilterCategory] = useState('__all__')
  const [filterLocation, setFilterLocation] = useState('__all__')
  const debouncedSearch = useDebounce(search)

  // Pagination
  const [page, setPage] = useState(1)

  // Dialogs
  const [formOpen, setFormOpen] = useState(false)
  const [editingItem, setEditingItem] = useState<Item | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<Item | null>(null)

  const locations = Array.from(new Set(items.map((i) => i.location).filter(Boolean)))

  const loadData = useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      const [itemRes, catRes] = await Promise.all([
        itemApi.list({
          category_id: filterCategory !== '__all__' ? filterCategory : undefined,
          location:    filterLocation !== '__all__' ? filterLocation : undefined,
        }),
        categoryApi.list(),
      ])
      setItems(itemRes.items ?? [])
      setCategories(catRes.categories ?? [])
      setPage(1)
    } catch (err) {
      setError(err instanceof Error ? err.message : '加载失败')
    } finally {
      setLoading(false)
    }
  }, [filterCategory, filterLocation])

  useEffect(() => { loadData() }, [loadData])

  // Reset page when search changes
  useEffect(() => { setPage(1) }, [debouncedSearch])

  // Client-side name filter → paginate
  const filteredItems = debouncedSearch
    ? items.filter((i) => i.name.toLowerCase().includes(debouncedSearch.toLowerCase()))
    : items

  const totalPages = Math.max(1, Math.ceil(filteredItems.length / PAGE_SIZE))
  const pagedItems = filteredItems.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE)

  // ── CRUD handlers ──────────────────────────────────────────────────────────

  function openCreate() { setEditingItem(null); setFormOpen(true) }
  function openEdit(item: Item) { setEditingItem(item); setFormOpen(true) }

  async function handleFormSubmit(payload: CreateItemPayload | UpdateItemPayload) {
    if (editingItem) {
      await itemApi.update(editingItem.id, payload as UpdateItemPayload)
    } else {
      await itemApi.create(payload as CreateItemPayload)
    }
    await loadData()
  }

  async function handleDelete() {
    if (!deleteTarget) return
    await itemApi.delete(deleteTarget.id)
    await loadData()
  }

  // ── Render ─────────────────────────────────────────────────────────────────

  return (
    <div className="flex flex-col h-full gap-6">
      {/* ── Toolbar ── */}
      <div className="flex items-center gap-3 flex-wrap">
        <div className="flex-1 min-w-0">
          <h1 className="text-2xl font-extrabold text-on-surface tracking-tight">物料库存</h1>
          <p className="text-xs text-on-surface-variant mt-0.5">管理家庭日常物资，精准掌握库存状态。</p>
        </div>
        <div className="flex items-center gap-2 flex-wrap">
          {/* Search */}
          <div className="relative min-w-48 max-w-64">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-on-surface-variant" />
            <Input className="pl-9" placeholder="搜索物料名称…" value={search} onChange={(e) => setSearch(e.target.value)} />
          </div>
          {/* Category filter */}
          <Select value={filterCategory} onValueChange={setFilterCategory}>
            <SelectTrigger className="w-34">
              <SelectValue placeholder="全部分类" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="__all__">全部分类</SelectItem>
              {categories.map((c) => <SelectItem key={c.id} value={c.id}>{c.name}</SelectItem>)}
            </SelectContent>
          </Select>
          {/* Location filter */}
          <Select value={filterLocation} onValueChange={setFilterLocation}>
            <SelectTrigger className="w-34">
              <SelectValue placeholder="全部位置" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="__all__">全部位置</SelectItem>
              {locations.map((loc) => <SelectItem key={loc} value={loc}>{loc}</SelectItem>)}
            </SelectContent>
          </Select>
          <Button variant="outline" size="icon" onClick={loadData} title="刷新">
            <RefreshCw className={cn('w-4 h-4', loading && 'animate-spin')} />
          </Button>
          <Button onClick={openCreate}>
            <Plus className="w-4 h-4" />
            新增物料
          </Button>
        </div>
      </div>

      {/* ── Error ── */}
      {error && (
        <div className="rounded-lg bg-error-container px-4 py-3 text-sm text-error">{error}</div>
      )}

      {/* ── Table card ── */}
      <div className="rounded-xl border border-outline-variant/20 bg-surface-container-lowest shadow-sm overflow-hidden">
        <Table>
          <TableHeader>
            <TableRow className="bg-surface-container-low/50 border-outline-variant/20 hover:bg-surface-container-low/50">
              <TableHead className="py-4 text-[10px] font-bold uppercase tracking-widest text-on-surface-variant w-50">物料名称</TableHead>
              <TableHead className="py-4 text-[10px] font-bold uppercase tracking-widest text-on-surface-variant w-28">分类</TableHead>
              <TableHead className="py-4 text-[10px] font-bold uppercase tracking-widest text-on-surface-variant w-25">库存数量</TableHead>
              <TableHead className="py-4 text-[10px] font-bold uppercase tracking-widest text-on-surface-variant w-28">存放位置</TableHead>
              <TableHead className="py-4 text-[10px] font-bold uppercase tracking-widest text-on-surface-variant w-28">购买日期</TableHead>
              <TableHead className="py-4 text-[10px] font-bold uppercase tracking-widest text-on-surface-variant w-28">过期日期</TableHead>
              <TableHead className="py-4 text-[10px] font-bold uppercase tracking-widest text-on-surface-variant w-25">状态</TableHead>
              <TableHead className="py-4 text-[10px] font-bold uppercase tracking-widest text-on-surface-variant w-20 text-right">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading && items.length === 0 ? (
              <TableRow>
                <TableCell colSpan={8} className="h-32 text-center text-on-surface-variant">加载中…</TableCell>
              </TableRow>
            ) : pagedItems.length === 0 ? (
              <TableRow>
                <TableCell colSpan={8} className="h-32 text-center text-on-surface-variant">暂无物料数据</TableCell>
              </TableRow>
            ) : (
              pagedItems.map((item) => {
                const status = getItemStatus(item)
                return (
                  <TableRow
                    key={item.id}
                    className="border-outline-variant/10 hover:bg-surface transition-colors group"
                  >
                    <TableCell className="font-bold text-on-surface py-5">{item.name}</TableCell>
                    <TableCell className="py-5">
                      {item.category?.name
                        ? <span className="px-2.5 py-1 bg-surface-container text-on-surface-variant text-xs font-semibold rounded-full">{item.category.name}</span>
                        : <span className="text-on-surface-variant text-sm">—</span>
                      }
                    </TableCell>
                    <TableCell className={cn('py-5 font-medium', status === 'expiring' && 'text-tertiary font-bold')}>
                      {item.quantity} {item.unit}
                    </TableCell>
                    <TableCell className="py-5 text-on-surface-variant">{item.location || '—'}</TableCell>
                    <TableCell className="py-5 text-on-surface-variant">{formatDate(item.purchased_at)}</TableCell>
                    <TableCell className={cn('py-5', status === 'expired' && 'text-error', status === 'expiring' && 'text-tertiary')}>
                      {item.expire_at ? formatDate(item.expire_at) : '—'}
                    </TableCell>
                    <TableCell className="py-5"><StatusBadge status={status} /></TableCell>
                    <TableCell className="py-5 text-right">
                      <div className="flex items-center justify-end gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                        <Button variant="ghost" size="icon" className="w-8 h-8" onClick={() => openEdit(item)} title="编辑">
                          <Pencil className="w-3.5 h-3.5" />
                        </Button>
                        <Button variant="ghost" size="icon" className="w-8 h-8 text-error hover:text-error hover:bg-error-container/20" onClick={() => setDeleteTarget(item)} title="删除">
                          <Trash2 className="w-3.5 h-3.5" />
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                )
              })
            )}
          </TableBody>
        </Table>

        {/* ── Pagination (inside card) ── */}
        {filteredItems.length > 0 && (
          <TablePagination
            page={page}
            totalPages={totalPages}
            totalItems={filteredItems.length}
            pageSize={PAGE_SIZE}
            onChange={setPage}
          />
        )}
      </div>

      {/* ── Summary cards ── */}
      <SummaryCards items={items} categories={categories} />

      {/* ── Dialogs ── */}
      <ItemFormDialog
        open={formOpen}
        item={editingItem}
        categories={categories}
        onClose={() => setFormOpen(false)}
        onSubmit={handleFormSubmit}
      />
      <DeleteConfirmDialog
        open={deleteTarget !== null}
        itemName={deleteTarget?.name ?? ''}
        onClose={() => setDeleteTarget(null)}
        onConfirm={handleDelete}
      />
    </div>
  )
}
