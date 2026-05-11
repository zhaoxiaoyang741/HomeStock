import { useMemo, useState } from 'react'
import { Pencil } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
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
import { cn } from '@/lib/utils'
import { formatDate } from '@/lib/format'
import type { StockLot } from '@/types/stock'
import { getInventoryStatus } from '@/types/material'

interface Props {
  open: boolean
  materialName: string
  lots: StockLot[]
  onClose: () => void
  onEditLot: (lot: StockLot) => void
  onAdjustLot: (lot: StockLot) => void
}

export default function LotDetailsDialog({ open, materialName, lots, onClose, onEditLot, onAdjustLot }: Props) {
  const { t, i18n } = useTranslation('inventory')
  const [showZeroStock, setShowZeroStock] = useState(false)

  const displayLots = useMemo(
    () => (showZeroStock ? lots : lots.filter((lot) => lot.quantity_on_hand > 0)),
    [lots, showZeroStock],
  )

  return (
    <Dialog open={open} onOpenChange={(next) => !next && onClose()}>
      <DialogContent className="sm:max-w-4xl">
        <DialogHeader>
          <DialogTitle>{t('lotDetails_title', { name: materialName })}</DialogTitle>
        </DialogHeader>

        {lots.some((lot) => lot.quantity_on_hand === 0) && (
          <label className="flex items-center gap-1.5 text-sm text-on-surface-variant cursor-pointer select-none px-1">
            <input
              type="checkbox"
              checked={showZeroStock}
              onChange={(e) => setShowZeroStock(e.target.checked)}
              className="w-4 h-4 rounded border-outline-variant accent-primary"
            />
            {t('lotShowZeroStock')}
          </label>
        )}

        <div className="overflow-auto max-h-[65vh]">
          <Table>
            <TableHeader>
              <TableRow className="border-outline-variant/20">
                <TableHead className="text-xs font-bold uppercase tracking-widest text-on-surface-variant">{t('lotsColStock')}</TableHead>
                <TableHead className="text-xs font-bold uppercase tracking-widest text-on-surface-variant">{t('lotsColLocation')}</TableHead>
                <TableHead className="text-xs font-bold uppercase tracking-widest text-on-surface-variant">{t('lotsColPurchased')}</TableHead>
                <TableHead className="text-xs font-bold uppercase tracking-widest text-on-surface-variant">{t('lotsColExpiry')}</TableHead>
                <TableHead className="text-xs font-bold uppercase tracking-widest text-on-surface-variant">{t('lotsColNotes')}</TableHead>
                <TableHead className="text-xs font-bold uppercase tracking-widest text-on-surface-variant">{t('lotsColStatus')}</TableHead>
                <TableHead className="text-xs font-bold uppercase tracking-widest text-on-surface-variant text-right">{t('lotsColActions')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {displayLots.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={7} className="h-32 text-center text-on-surface-variant">
                    {t('lotsEmpty')}
                  </TableCell>
                </TableRow>
              ) : (
                displayLots.map((lot) => {
                  const status = getInventoryStatus(lot.expire_at)
                  return (
                    <TableRow key={lot.id} className="border-outline-variant/10 hover:bg-surface">
                      <TableCell className="py-4 font-medium">
                        <div className="flex items-center gap-2">
                          <span className={cn(lot.quantity_on_hand === 0 && 'text-on-surface-variant')}>{lot.quantity_on_hand} {lot.unit}</span>
                          {lot.quantity_on_hand === 0 && (
                            <Badge variant="outline" className="text-xs text-on-surface-variant border-outline-variant/50">{t('statusZero')}</Badge>
                          )}
                        </div>
                      </TableCell>
                      <TableCell className="py-4 text-on-surface-variant">{lot.location || '—'}</TableCell>
                      <TableCell className="py-4 text-on-surface-variant">
                        {lot.purchased_at ? formatDate(lot.purchased_at, i18n.language) : '—'}
                      </TableCell>
                      <TableCell className="py-4 text-on-surface-variant">
                        {lot.expire_at ? formatDate(lot.expire_at, i18n.language) : '—'}
                      </TableCell>
                      <TableCell className="py-4 text-on-surface-variant">{lot.notes || '—'}</TableCell>
                      <TableCell className="py-4">
                        <Badge
                          variant={status === 'expired' ? 'destructive' : status === 'expiring' ? 'outline' : 'default'}
                          className={cn(status === 'expiring' && 'text-tertiary border-tertiary/50')}
                        >
                          {status === 'normal' ? t('statusNormal') : status === 'expiring' ? t('statusExpiring') : t('statusExpired')}
                        </Badge>
                      </TableCell>
                      <TableCell className="py-4 text-right">
                        <div className="flex items-center justify-end gap-1">
                          <Button variant="ghost" size="icon" onClick={() => onEditLot(lot)} title={t('btnEditLot')}>
                            <Pencil className="w-4 h-4" />
                          </Button>
                          <Button variant="outline" size="sm" onClick={() => onAdjustLot(lot)}>
                            {t('btnAdjustLot')}
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
      </DialogContent>
    </Dialog>
  )
}
