import { useCallback, useEffect, useMemo, useState } from 'react'
import {
  Eye,
  PackageMinus,
  PackagePlus,
  RefreshCw,
  Search,
} from 'lucide-react'
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
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  Pagination,
  PaginationContent,
  PaginationItem,
  PaginationLink,
  PaginationNext,
  PaginationPrevious,
} from '@/components/ui/pagination'
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
import EditLotDialog from './EditLotDialog'
import AdjustLotDialog from './AdjustLotDialog'
import ExpiringLotsDialog from './ExpiringLotsDialog'
import LotDetailsDialog from './LotDetailsDialog'
import ConsumeMaterialDialog from './ConsumeMaterialDialog'

function formatLocations(locations: string[]): string {
  if (!locations ||  locations.length === 0) return '—'
  if (locations.length <= 2) return locations.join(' / ')
  return `${locations.slice(0, 2).join(' / ')} +${locations.length - 2}`
}

export default function InventoryPage() {
  const { t, i18n } = useTranslation('inventory')
  const [materials, setMaterials] = useState<MaterialSummary[]>([])
  const [lots, setLots] = useState<StockLot[]>([])
  const [categories, setCategories] = useState<Category[]>([])
  const [selectedMaterialId, setSelectedMaterialId] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  const [search, setSearch] = useState(() => {
    const params = new URLSearchParams(window.location.search)
    return params.get('keyword') || ''
  })
  const [categoryFilter, setCategoryFilter] = useState('__all__')
  const [locationFilter, setLocationFilter] = useState('__all__')
  const debouncedSearch = useDebounce(search, 300)

  const [expiringOpen, setExpiringOpen] = useState(false)
  const [inboundOpen, setInboundOpen] = useState(false)
  const [editingLot, setEditingLot] = useState<StockLot | null>(null)
  const [adjustingLot, setAdjustingLot] = useState<StockLot | null>(null)
  const [lotDetailsOpen, setLotDetailsOpen] = useState(false)
  const [consumingMaterial, setConsumingMaterial] = useState<MaterialSummary | null>(null)
  const [showZeroStock, setShowZeroStock] = useState(false)

  const [page, setPage] = useState(1)
  const pageSize = 10
  const totalPages = useMemo(
    () => Math.max(1, Math.ceil(materials.length / pageSize)),
    [materials.length],
  )
  const paginatedMaterials = useMemo(
    () => materials.slice((page - 1) * pageSize, page * pageSize),
    [materials, page, pageSize],
  )

  useEffect(() => {
    setPage(1)
  }, [materials.length])

  const STATUS_MAP: Record<InventoryStatus, { label: string; variant: 'default' | 'outline' | 'destructive' }> = {
    normal: { label: t('statusNormal'), variant: 'default' },
    expiring: { label: t('statusExpiring'), variant: 'outline' },
    expired: { label: t('statusExpired'), variant: 'destructive' },
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

  const loadData = useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      const [materialRes, lotRes, categoryRes] = await Promise.all([
        materialApi.list({
          category_id: categoryFilter !== '__all__' ? categoryFilter : undefined,
          keyword: debouncedSearch || undefined,
          show_zero_stock: showZeroStock || undefined,
        }),
        stockLotApi.list({
          category_id: categoryFilter !== '__all__' ? categoryFilter : undefined,
          keyword: debouncedSearch || undefined,
          location: locationFilter !== '__all__' ? locationFilter : undefined,
          show_zero_stock: showZeroStock || undefined,
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
      setError(err instanceof Error ? err.message : t('loadingFailed'))
    } finally {
      setLoading(false)
    }
  }, [categoryFilter, locationFilter, debouncedSearch, showZeroStock, t])

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

  async function handleInbound(payload: InboundStockLotPayload) {
    await stockLotApi.inbound(payload)
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

  async function handleConsume(payload: ConsumeMaterialPayload) {
    if (!consumingMaterial) return
    await materialApi.consume(consumingMaterial.id, payload)
    await loadData()
    setConsumingMaterial(null)
  }

  return (
    <div className="flex flex-col h-full gap-6">
      <div className="flex items-center gap-3 flex-wrap">
        <div className="flex-1 min-w-0">
          <h1 className="text-2xl font-extrabold text-on-surface tracking-tight">{t('title')}</h1>
          <p className="text-sm text-on-surface-variant mt-0.5">{t('subtitle')}</p>
        </div>
        <div className="flex items-center gap-2 flex-wrap">
          <div className="relative min-w-56">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-on-surface-variant" />
            <Input
              className="pl-9"
              value={search}
              onChange={(event) => setSearch(event.target.value)}
              placeholder={t('searchPlaceholder')}
            />
          </div>
          <Select value={categoryFilter} onValueChange={setCategoryFilter}>
            <SelectTrigger className="w-40">
              <SelectValue placeholder={t('allCategories')} />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="__all__">{t('allCategories')}</SelectItem>
              {categories.map((category) => (
                <SelectItem key={category.id} value={category.id}>
                  {category.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Select value={locationFilter} onValueChange={setLocationFilter}>
            <SelectTrigger className="w-40">
              <SelectValue placeholder={t('allLocations')} />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="__all__">{t('allLocations')}</SelectItem>
              {locationOptions.map((location) => (
                <SelectItem key={location} value={location}>
                  {location}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <label className="flex items-center gap-1.5 text-sm text-on-surface-variant cursor-pointer select-none">
            <input
              type="checkbox"
              checked={showZeroStock}
              onChange={(e) => setShowZeroStock(e.target.checked)}
              className="w-4 h-4 rounded border-outline-variant accent-primary"
            />
            {t('showZeroStock')}
          </label>
          <Button variant="outline" size="icon" onClick={() => void loadData()} title={t('common:refresh')}>
            <RefreshCw className={cn('w-4 h-4', loading && 'animate-spin')} />
          </Button>
          <Button onClick={() => setInboundOpen(true)}>
            <PackagePlus className="w-4 h-4" />
            {t('btnInbound')}
          </Button>
        </div>
      </div>

      {error && (
        <div className="rounded-lg bg-error-container px-4 py-3 text-sm text-error">{error}</div>
      )}

      <div className="flex-1 flex flex-col gap-4 min-h-0">
        <div className="flex-1 rounded-xl border border-outline-variant/20 bg-surface-container-lowest shadow-sm overflow-hidden">
          <div className="h-full overflow-auto">
        <Table>
          <TableHeader>
            <TableRow className="bg-surface-container-low/50 border-outline-variant/20 hover:bg-surface-container-low/50">
              <TableHead className="py-4 text-xs font-bold uppercase tracking-widest text-on-surface-variant">{t('colMaterial')}</TableHead>
              <TableHead className="py-4 text-xs font-bold uppercase tracking-widest text-on-surface-variant">{t('colCategory')}</TableHead>
              <TableHead className="py-4 text-xs font-bold uppercase tracking-widest text-on-surface-variant">{t('colTotalStock')}</TableHead>
              <TableHead className="py-4 text-xs font-bold uppercase tracking-widest text-on-surface-variant">{t('colLotCount')}</TableHead>
              <TableHead className="py-4 text-xs font-bold uppercase tracking-widest text-on-surface-variant">{t('colNearestExpiry')}</TableHead>
              <TableHead className="py-4 text-xs font-bold uppercase tracking-widest text-on-surface-variant">{t('colLocations')}</TableHead>
              <TableHead className="py-4 text-xs font-bold uppercase tracking-widest text-on-surface-variant">{t('colStatus')}</TableHead>
              <TableHead className="py-4 text-xs font-bold uppercase tracking-widest text-on-surface-variant text-right">{t('colActions')}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {materials.length === 0 ? (
              <TableRow>
                <TableCell colSpan={8} className="h-32 text-center text-on-surface-variant">
                  {loading ? t('common:loading') : t('noMaterials')}
                </TableCell>
              </TableRow>
            ) : (
              paginatedMaterials.map((material) => {
                const status = getInventoryStatus(material.nearest_expire_at)
                return (
                  <TableRow
                    key={material.id}
                    className={cn('border-outline-variant/10 hover:bg-surface transition-colors', material.total_quantity === 0 && 'opacity-50')}
                  >
                    <TableCell className="py-5">
                      <div className="font-bold text-on-surface">{material.name}</div>
                      <div className="text-xs text-on-surface-variant">{material.spec || t('common:noSpec')}</div>
                    </TableCell>
                    <TableCell className="py-5 text-on-surface-variant">{material.category?.name ?? '-'}</TableCell>
                    <TableCell className="py-5 font-medium"><div className="flex items-center gap-2">{material.total_quantity} {material.default_unit}{material.total_quantity === 0 && <Badge variant="outline" className="text-xs text-on-surface-variant border-outline-variant/50">{t('statusZero')}</Badge>}</div></TableCell>
                    <TableCell className="py-5 text-on-surface-variant">{material.lot_count}</TableCell>
                    <TableCell className="py-5 text-on-surface-variant">
                      {material.nearest_expire_at ? formatDate(material.nearest_expire_at, i18n.language) : '-'}
                    </TableCell>
                    <TableCell className="py-5 text-on-surface-variant">{formatLocations(material.locations)}</TableCell>
                    <TableCell className="py-5">
                      <StatusBadge status={status} />
                    </TableCell>
                    <TableCell className="py-5 text-right">
                      <div className="flex items-center justify-end gap-1">
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => setConsumingMaterial(material)}
                          title={t('btnConsume')}
                        >
                          <PackageMinus className="w-4 h-4" />
                        </Button>
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => {
                            setSelectedMaterialId(material.id)
                            setLotDetailsOpen(true)
                          }}
                        >
                          <Eye className="w-4 h-4" />
                          {t('btnViewDetails')}
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
      </div>

      <div className="flex justify-center">
          <Pagination>
            <PaginationContent>
              <PaginationItem>
                <PaginationPrevious
                  onClick={() => setPage((p) => Math.max(1, p - 1))}
                  className={cn(page <= 1 && 'pointer-events-none opacity-50')}
                >
                  {t('paginationPrevious')}
                </PaginationPrevious>
              </PaginationItem>
              {Array.from({ length: totalPages }, (_, i) => i + 1).map((num) => (
                <PaginationItem key={num}>
                  <PaginationLink isActive={num === page} onClick={() => setPage(num)}>
                    {num}
                  </PaginationLink>
                </PaginationItem>
              ))}
              <PaginationItem>
                <PaginationNext
                  onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                  className={cn(page >= totalPages && 'pointer-events-none opacity-50')}
                >
                  {t('paginationNext')}
                </PaginationNext>
              </PaginationItem>
            </PaginationContent>
          </Pagination>
        </div>
      </div>

      <ExpiringLotsDialog
        open={expiringOpen}
        lots={lots.filter((lot) => lot.quantity_on_hand > 0 && getInventoryStatus(lot.expire_at) !== 'normal')}
        onClose={() => setExpiringOpen(false)}
        onChanged={() => void loadData()}
      />
      <InboundLotDialog
        open={inboundOpen}
        categories={categories}
        initialName={selectedMaterial?.name}
        initialSpec={selectedMaterial?.spec}
        onClose={() => setInboundOpen(false)}
        onSubmit={handleInbound}
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
      <LotDetailsDialog
        open={lotDetailsOpen}
        materialName={selectedMaterial?.name ?? ''}
        lots={selectedLots}
        onClose={() => setLotDetailsOpen(false)}
        onEditLot={setEditingLot}
        onAdjustLot={setAdjustingLot}
        onConsume={() => {
          if (selectedMaterial) setConsumingMaterial(selectedMaterial)
        }}
      />
      <ConsumeMaterialDialog
        open={consumingMaterial !== null}
        material={consumingMaterial}
        onClose={() => setConsumingMaterial(null)}
        onSubmit={handleConsume}
      />
    </div>
  )
}

