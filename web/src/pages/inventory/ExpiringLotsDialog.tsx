import { useEffect, useState } from 'react'
import { Pencil, Trash2, Check, X } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
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
import { stockLotApi } from '@/api/stockLot'
import type { StockLot, UpdateStockLotPayload } from '@/types/stock'
import { getInventoryStatus } from '@/types/material'

interface Props {
  open: boolean
  lots: StockLot[]
  onClose: () => void
  onChanged: () => void
}

function toDateInput(value: string | null): string {
  return value ? value.slice(0, 10) : ''
}

function toRFC3339(dateInput: string): string {
  return dateInput ? new Date(dateInput).toISOString() : ''
}

export default function ExpiringLotsDialog({ open, lots, onClose, onChanged }: Props) {
  const { t, i18n } = useTranslation('inventory')
  const [editingId, setEditingId] = useState<string | null>(null)
  const [editExpireAt, setEditExpireAt] = useState('')
  const [saving, setSaving] = useState(false)
  const [voidingId, setVoidingId] = useState<string | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    if (!open) {
      setEditingId(null)
      setError('')
    }
  }, [open])

  function startEdit(lot: StockLot) {
    setEditingId(lot.id)
    setEditExpireAt(toDateInput(lot.expire_at))
    setError('')
  }

  function cancelEdit() {
    setEditingId(null)
    setError('')
  }

  async function saveExpireAt(lot: StockLot) {
    setSaving(true)
    setError('')
    try {
      const payload: UpdateStockLotPayload = {
        expire_at: editExpireAt ? toRFC3339(editExpireAt) : '',
      }
      await stockLotApi.update(lot.id, payload)
      setEditingId(null)
      onChanged()
    } catch (err) {
      setError(err instanceof Error ? err.message : t('expiring_errorUpdateFailed'))
    } finally {
      setSaving(false)
    }
  }

  async function handleVoid(lot: StockLot) {
    setVoidingId(lot.id)
    setError('')
    try {
      await stockLotApi.void(lot.id)
      onChanged()
    } catch (err) {
      setError(err instanceof Error ? err.message : t('expiring_errorVoidFailed'))
    } finally {
      setVoidingId(null)
    }
  }

  return (
    <Dialog open={open} onOpenChange={(next) => !next && onClose()}>
      <DialogContent className="sm:max-w-3xl">
        <DialogHeader>
          <DialogTitle>{t('expiring_title')}</DialogTitle>
          <DialogDescription>
            {t('expiring_description')}
          </DialogDescription>
        </DialogHeader>

        {error && (
          <div className="rounded-lg bg-error-container px-3 py-2 text-sm text-error">{error}</div>
        )}

        <div className="overflow-auto max-h-[60vh]">
          <Table>
            <TableHeader>
              <TableRow className="bg-surface-container-low/50 border-outline-variant/20 hover:bg-surface-container-low/50">
                <TableHead className="text-xs font-bold uppercase tracking-widest text-on-surface-variant">{t('expiring_colMaterial')}</TableHead>
                <TableHead className="text-xs font-bold uppercase tracking-widest text-on-surface-variant">{t('expiring_colStock')}</TableHead>
                <TableHead className="text-xs font-bold uppercase tracking-widest text-on-surface-variant">{t('expiring_colLocation')}</TableHead>
                <TableHead className="text-xs font-bold uppercase tracking-widest text-on-surface-variant">{t('expiring_colExpiry')}</TableHead>
                <TableHead className="text-xs font-bold uppercase tracking-widest text-on-surface-variant">{t('expiring_colStatus')}</TableHead>
                <TableHead className="text-xs font-bold uppercase tracking-widest text-on-surface-variant text-right">{t('expiring_colActions')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {lots.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={6} className="h-24 text-center text-on-surface-variant">
                    {t('expiring_noLots')}
                  </TableCell>
                </TableRow>
              ) : (
                lots.map((lot) => {
                  const status = getInventoryStatus(lot.expire_at)
                  const isEditing = editingId === lot.id
                  const isVoiding = voidingId === lot.id
                  return (
                    <TableRow key={lot.id} className="border-outline-variant/10">
                      <TableCell className="py-4">
                        <div className="font-medium text-on-surface">{lot.material?.name ?? '—'}</div>
                        <div className="text-xs text-on-surface-variant">{lot.material?.spec || ''}</div>
                      </TableCell>
                      <TableCell className="py-4 text-on-surface-variant">
                        {lot.quantity_on_hand} {lot.unit}
                      </TableCell>
                      <TableCell className="py-4 text-on-surface-variant">{lot.location || '—'}</TableCell>
                      <TableCell className="py-4">
                        {isEditing ? (
                          <Input
                            type="date"
                            value={editExpireAt}
                            onChange={(e) => setEditExpireAt(e.target.value)}
                            className="w-36 h-8 text-sm"
                            autoFocus
                          />
                        ) : (
                          <span className={cn(
                            'text-sm',
                            status === 'expired' ? 'text-error' : 'text-tertiary'
                          )}>
                            {lot.expire_at ? formatDate(lot.expire_at, i18n.language) : '—'}
                          </span>
                        )}
                      </TableCell>
                      <TableCell className="py-4">
                        <Badge
                          variant={status === 'expired' ? 'destructive' : 'outline'}
                          className={cn(status === 'expiring' && 'text-tertiary border-tertiary/50')}
                        >
                          {status === 'expired' ? t('expiring_statusExpired') : t('expiring_statusExpiring')}
                        </Badge>
                      </TableCell>
                      <TableCell className="py-4 text-right">
                        <div className="flex items-center justify-end gap-1">
                          {isEditing ? (
                            <>
                              <Button
                                variant="ghost"
                                size="icon"
                                onClick={() => void saveExpireAt(lot)}
                                disabled={saving}
                                title={t('expiring_tooltipConfirm')}
                              >
                                <Check className="w-4 h-4 text-primary" />
                              </Button>
                              <Button variant="ghost" size="icon" onClick={cancelEdit} disabled={saving} title={t('expiring_tooltipCancel')}>
                                <X className="w-4 h-4" />
                              </Button>
                            </>
                          ) : (
                            <>
                              <Button
                                variant="ghost"
                                size="icon"
                                onClick={() => startEdit(lot)}
                                disabled={isVoiding}
                                title={t('expiring_tooltipEdit')}
                              >
                                <Pencil className="w-4 h-4" />
                              </Button>
                              <Button
                                variant="ghost"
                                size="icon"
                                onClick={() => void handleVoid(lot)}
                                disabled={isVoiding}
                                title={t('expiring_tooltipVoid')}
                              >
                                <Trash2 className={cn('w-4 h-4 text-error', isVoiding && 'opacity-50')} />
                              </Button>
                            </>
                          )}
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
