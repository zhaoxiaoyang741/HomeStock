import { useCallback, useEffect, useMemo, useState } from 'react'
import {
  Box,
  Boxes,
  ChevronRight,
  Pencil,
  PackagePlus,
  RefreshCw,
  Search,
  SlidersHorizontal,
  Warehouse,
} from 'lucide-react'
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
import { categoryApi } from '@/api/category'
import { materialApi } from '@/api/material'
import { stockLotApi } from '@/api/stockLot'
import type { Category } from '@/types/category'
import type { InboundStockLotPayload, StockLot, UpdateStockLotPayload, AdjustStockLotPayload } from '@/types/stock'
import type { ConsumeMaterialPayload, MaterialSummary, InventoryStatus } from '@/types/material'
import { getInventoryStatus } from '@/types/material'
import InboundLotDialog from './InboundLotDialog'
import ConsumeMaterialDialog from './ConsumeMaterialDialog'
import EditLotDialog from './EditLotDialog'
import AdjustLotDialog from './AdjustLotDialog'

const STATUS_MAP: Record<InventoryStatus, { label: string; variant: 'default' | 'outline' | 'destructive' }> = {
  normal: { label: '正常', variant: 'default' },
  expiring: { label: '即将过期', variant: 'outline' },
  expired: { label: '已过期', variant: 'destructive' },
}

function StatusBadge({ status }: { status: InventoryStatus }) {
  const meta = STATUS_MAP[status]
  return (
    <Badge
      variant={meta.variant}
      className={cn(meta.variant === 'outline' && 'text-tertiary border-tertiary/50')}
    >
      {meta.label}
    </Badge>
  )
}

function formatLocations(locations: string[]): string {
  if (locations.length === 0) return '—'
  if (locations.length <= 2) return locations.join(' / ')
  return `${locations.slice(0, 2).join(' / ')} +${locations.length - 2}`
}

export default function InventoryPage() {
  const [materials, setMaterials] = useState<MaterialSummary[]>([])
  const [lots, setLots] = useState<StockLot[]>([])
  const [categories, setCategories] = useState<Category[]>([])
  const [selectedMaterialId, setSelectedMaterialId] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  const [search, setSearch] = useState('')
  const [categoryFilter, setCategoryFilter] = useState('__all__')
  const [locationFilter, setLocationFilter] = useState('__all__')
  const debouncedSearch = useDebounce(search, 300)

  const [inboundOpen, setInboundOpen] = useState(false)
  const [consumeOpen, setConsumeOpen] = useState(false)
  const [editingLot, setEditingLot] = useState<StockLot | null>(null)
  const [adjustingLot, setAdjustingLot] = useState<StockLot | null>(null)

  const loadData = useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      const [materialRes, lotRes, categoryRes] = await Promise.all([
        materialApi.list({
          category_id: categoryFilter !== '__all__' ? categoryFilter : undefined,
          keyword: debouncedSearch || undefined,
        }),
        stockLotApi.list({
          category_id: categoryFilter !== '__all__' ? categoryFilter : undefined,
          keyword: debouncedSearch || undefined,
          location: locationFilter !== '__all__' ? locationFilter : undefined,
        }),
        categoryApi.list(),
      ])

      const lotMaterialIDs = new Set(lotRes.lots.map((lot) => lot.material_id))
      const nextMaterials =
        locationFilter === '__all__'
          ? materialRes.materials
          : materialRes.materials.filter((material) => lotMaterialIDs.has(material.id))

      setMaterials(nextMaterials)
      setLots(lotRes.lots)
      setCategories(categoryRes.categories)
      setSelectedMaterialId((current) =>
        nextMaterials.some((material) => material.id === current) ? current : (nextMaterials[0]?.id ?? ''),
      )
    } catch (err) {
      setError(err instanceof Error ? err.message : '加载库存失败')
    } finally {
      setLoading(false)
    }
  }, [categoryFilter, locationFilter, debouncedSearch])

  useEffect(() => {
    void loadData()
  }, [loadData])

  const locationOptions = useMemo(
    () => Array.from(new Set(lots.map((lot) => lot.location).filter(Boolean))).sort((left, right) => left.localeCompare(right)),
    [lots],
  )

  const selectedMaterial = useMemo(
    () => materials.find((material) => material.id === selectedMaterialId) ?? null,
    [materials, selectedMaterialId],
  )

  const selectedLots = useMemo(
    () => lots.filter((lot) => lot.material_id === selectedMaterialId),
    [lots, selectedMaterialId],
  )

  const warningLots = useMemo(
    () => lots.filter((lot) => getInventoryStatus(lot.expire_at) !== 'normal').length,
    [lots],
  )

  async function handleInbound(payload: InboundStockLotPayload) {
    await stockLotApi.inbound(payload)
    await loadData()
  }

  async function handleConsume(payload: ConsumeMaterialPayload) {
    if (!selectedMaterial) return
    await materialApi.consume(selectedMaterial.id, payload)
    await loadData()
  }

  async function handleUpdateLot(payload: UpdateStockLotPayload) {
    if (!editingLot) return
    await stockLotApi.update(editingLot.id, payload)
    await loadData()
  }

  async function handleAdjustLot(payload: AdjustStockLotPayload) {
    if (!adjustingLot) return
    await stockLotApi.adjust(adjustingLot.id, payload)
    await loadData()
  }

  return (
    <div className="flex flex-col h-full gap-6">
      <div className="flex items-center gap-3 flex-wrap">
        <div className="flex-1 min-w-0">
          <h1 className="text-2xl font-extrabold text-on-surface tracking-tight">物料库存</h1>
          <p className="text-sm text-on-surface-variant mt-0.5">按物料汇总库存，并在下方查看每个真实批次的库存、日期和位置。</p>
        </div>
        <div className="flex items-center gap-2 flex-wrap">
          <div className="relative min-w-56">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-on-surface-variant" />
            <Input
              className="pl-9"
              value={search}
              onChange={(event) => setSearch(event.target.value)}
              placeholder="搜索物料名称或规格"
            />
          </div>
          <Select value={categoryFilter} onValueChange={setCategoryFilter}>
            <SelectTrigger className="w-40">
              <SelectValue placeholder="全部分类" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="__all__">全部分类</SelectItem>
              {categories.map((category) => (
                <SelectItem key={category.id} value={category.id}>
                  {category.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Select value={locationFilter} onValueChange={setLocationFilter}>
            <SelectTrigger className="w-40">
              <SelectValue placeholder="全部位置" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="__all__">全部位置</SelectItem>
              {locationOptions.map((location) => (
                <SelectItem key={location} value={location}>
                  {location}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Button variant="outline" size="icon" onClick={() => void loadData()} title="刷新">
            <RefreshCw className={cn('w-4 h-4', loading && 'animate-spin')} />
          </Button>
          <Button variant="outline" onClick={() => setConsumeOpen(true)} disabled={!selectedMaterial}>
            <SlidersHorizontal className="w-4 h-4" />
            消耗库存
          </Button>
          <Button onClick={() => setInboundOpen(true)}>
            <PackagePlus className="w-4 h-4" />
            新增入库
          </Button>
        </div>
      </div>

      {error && (
        <div className="rounded-lg bg-error-container px-4 py-3 text-sm text-error">{error}</div>
      )}

      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div className="p-5 bg-surface-container-lowest rounded-xl border border-outline-variant/20 flex items-center gap-4 shadow-sm">
          <div className="w-11 h-11 rounded-full bg-primary/10 flex items-center justify-center shrink-0">
            <Warehouse className="w-5 h-5 text-primary" />
          </div>
          <div>
            <p className="text-xs font-bold uppercase tracking-widest text-on-surface-variant">物料数</p>
            <p className="text-xl font-extrabold text-on-surface">{materials.length}</p>
          </div>
        </div>
        <div className="p-5 bg-surface-container-lowest rounded-xl border border-outline-variant/20 flex items-center gap-4 shadow-sm">
          <div className="w-11 h-11 rounded-full bg-secondary-container/20 flex items-center justify-center shrink-0">
            <Boxes className="w-5 h-5 text-secondary" />
          </div>
          <div>
            <p className="text-xs font-bold uppercase tracking-widest text-on-surface-variant">批次数</p>
            <p className="text-xl font-extrabold text-on-surface">{lots.length}</p>
          </div>
        </div>
        <div className="p-5 bg-surface-container-lowest rounded-xl border border-outline-variant/20 flex items-center gap-4 shadow-sm">
          <div className="w-11 h-11 rounded-full bg-tertiary-container/20 flex items-center justify-center shrink-0">
            <Box className="w-5 h-5 text-tertiary" />
          </div>
          <div>
            <p className="text-xs font-bold uppercase tracking-widest text-on-surface-variant">临期批次</p>
            <p className={cn('text-xl font-extrabold', warningLots > 0 ? 'text-tertiary' : 'text-on-surface')}>
              {warningLots}
            </p>
          </div>
        </div>
      </div>

      <div className="rounded-xl border border-outline-variant/20 bg-surface-container-lowest shadow-sm overflow-hidden">
        <Table>
          <TableHeader>
            <TableRow className="bg-surface-container-low/50 border-outline-variant/20 hover:bg-surface-container-low/50">
              <TableHead className="py-4 text-xs font-bold uppercase tracking-widest text-on-surface-variant">物料名称</TableHead>
              <TableHead className="py-4 text-xs font-bold uppercase tracking-widest text-on-surface-variant">分类</TableHead>
              <TableHead className="py-4 text-xs font-bold uppercase tracking-widest text-on-surface-variant">总库存</TableHead>
              <TableHead className="py-4 text-xs font-bold uppercase tracking-widest text-on-surface-variant">批次数</TableHead>
              <TableHead className="py-4 text-xs font-bold uppercase tracking-widest text-on-surface-variant">最近过期</TableHead>
              <TableHead className="py-4 text-xs font-bold uppercase tracking-widest text-on-surface-variant">位置摘要</TableHead>
              <TableHead className="py-4 text-xs font-bold uppercase tracking-widest text-on-surface-variant">状态</TableHead>
              <TableHead className="py-4 text-xs font-bold uppercase tracking-widest text-on-surface-variant text-right">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {materials.length === 0 ? (
              <TableRow>
                <TableCell colSpan={8} className="h-32 text-center text-on-surface-variant">
                  {loading ? '加载中…' : '暂无物料'}
                </TableCell>
              </TableRow>
            ) : (
              materials.map((material) => {
                const status = getInventoryStatus(material.nearest_expire_at)
                const selected = material.id === selectedMaterialId
                return (
                  <TableRow
                    key={material.id}
                    className={cn(
                      'border-outline-variant/10 cursor-pointer hover:bg-surface transition-colors',
                      selected && 'bg-primary/5',
                    )}
                    onClick={() => setSelectedMaterialId(material.id)}
                  >
                    <TableCell className="py-5">
                      <div className="font-bold text-on-surface">{material.name}</div>
                      <div className="text-xs text-on-surface-variant">{material.spec || '未填写规格'}</div>
                    </TableCell>
                    <TableCell className="py-5 text-on-surface-variant">{material.category?.name ?? '—'}</TableCell>
                    <TableCell className="py-5 font-medium">{material.total_quantity} {material.default_unit}</TableCell>
                    <TableCell className="py-5 text-on-surface-variant">{material.lot_count}</TableCell>
                    <TableCell className="py-5 text-on-surface-variant">
                      {material.nearest_expire_at ? formatDate(material.nearest_expire_at) : '—'}
                    </TableCell>
                    <TableCell className="py-5 text-on-surface-variant">{formatLocations(material.locations)}</TableCell>
                    <TableCell className="py-5">
                      <StatusBadge status={status} />
                    </TableCell>
                    <TableCell className="py-5 text-right">
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={(event) => {
                          event.stopPropagation()
                          setSelectedMaterialId(material.id)
                          setConsumeOpen(true)
                        }}
                        title="消耗库存"
                      >
                        <ChevronRight className="w-4 h-4" />
                      </Button>
                    </TableCell>
                  </TableRow>
                )
              })
            )}
          </TableBody>
        </Table>
      </div>

      <div className="rounded-xl border border-outline-variant/20 bg-surface-container-lowest shadow-sm overflow-hidden">
        <div className="px-6 py-5 border-b border-outline-variant/20 flex items-center justify-between gap-4">
          <div>
            <h2 className="text-lg font-semibold text-on-surface">批次明细</h2>
            <p className="text-sm text-on-surface-variant mt-0.5">
              {selectedMaterial
                ? `${selectedMaterial.name}${selectedMaterial.spec ? ` / ${selectedMaterial.spec}` : ''} 的真实库存批次`
                : '选择上方物料后查看批次明细'}
            </p>
          </div>
          {selectedMaterial && (
            <div className="text-right">
              <p className="text-xs font-bold uppercase tracking-widest text-on-surface-variant">当前汇总</p>
              <p className="text-sm font-semibold text-on-surface">
                {selectedMaterial.total_quantity} {selectedMaterial.default_unit} / {selectedMaterial.lot_count} 批
              </p>
            </div>
          )}
        </div>

        <Table>
          <TableHeader>
            <TableRow className="border-outline-variant/20">
              <TableHead className="text-xs font-bold uppercase tracking-widest text-on-surface-variant">库存</TableHead>
              <TableHead className="text-xs font-bold uppercase tracking-widest text-on-surface-variant">位置</TableHead>
              <TableHead className="text-xs font-bold uppercase tracking-widest text-on-surface-variant">购买日期</TableHead>
              <TableHead className="text-xs font-bold uppercase tracking-widest text-on-surface-variant">过期日期</TableHead>
              <TableHead className="text-xs font-bold uppercase tracking-widest text-on-surface-variant">备注</TableHead>
              <TableHead className="text-xs font-bold uppercase tracking-widest text-on-surface-variant">状态</TableHead>
              <TableHead className="text-xs font-bold uppercase tracking-widest text-on-surface-variant text-right">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {!selectedMaterial ? (
              <TableRow>
                <TableCell colSpan={7} className="h-32 text-center text-on-surface-variant">
                  请先在上方选择一个物料
                </TableCell>
              </TableRow>
            ) : selectedLots.length === 0 ? (
              <TableRow>
                <TableCell colSpan={7} className="h-32 text-center text-on-surface-variant">
                  当前筛选条件下没有批次
                </TableCell>
              </TableRow>
            ) : (
              selectedLots.map((lot) => {
                const status = getInventoryStatus(lot.expire_at)
                return (
                  <TableRow key={lot.id} className="border-outline-variant/10 hover:bg-surface">
                    <TableCell className="py-5 font-medium">{lot.quantity_on_hand} {lot.unit}</TableCell>
                    <TableCell className="py-5 text-on-surface-variant">{lot.location || '—'}</TableCell>
                    <TableCell className="py-5 text-on-surface-variant">
                      {lot.purchased_at ? formatDate(lot.purchased_at) : '—'}
                    </TableCell>
                    <TableCell className="py-5 text-on-surface-variant">
                      {lot.expire_at ? formatDate(lot.expire_at) : '—'}
                    </TableCell>
                    <TableCell className="py-5 text-on-surface-variant">{lot.notes || '—'}</TableCell>
                    <TableCell className="py-5">
                      <StatusBadge status={status} />
                    </TableCell>
                    <TableCell className="py-5 text-right">
                      <div className="flex items-center justify-end gap-1">
                        <Button variant="ghost" size="icon" onClick={() => setEditingLot(lot)} title="编辑批次">
                          <Pencil className="w-4 h-4" />
                        </Button>
                        <Button variant="outline" size="sm" onClick={() => setAdjustingLot(lot)}>
                          调整
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                )
              })
            )}
          </TableBody>
        </Table>
      </div>

      <InboundLotDialog
        open={inboundOpen}
        categories={categories}
        initialName={selectedMaterial?.name}
        initialSpec={selectedMaterial?.spec}
        onClose={() => setInboundOpen(false)}
        onSubmit={handleInbound}
      />
      <ConsumeMaterialDialog
        open={consumeOpen}
        material={selectedMaterial}
        onClose={() => setConsumeOpen(false)}
        onSubmit={handleConsume}
      />
      <EditLotDialog
        open={editingLot !== null}
        lot={editingLot}
        onClose={() => setEditingLot(null)}
        onSubmit={handleUpdateLot}
      />
      <AdjustLotDialog
        open={adjustingLot !== null}
        lot={adjustingLot}
        onClose={() => setAdjustingLot(null)}
        onSubmit={handleAdjustLot}
      />
    </div>
  )
}
