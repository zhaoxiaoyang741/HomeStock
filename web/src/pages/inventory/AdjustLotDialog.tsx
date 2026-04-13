import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import type { AdjustStockLotPayload, StockLot } from '@/types/stock'

interface Props {
  open: boolean
  lot: StockLot | null
  onClose: () => void
  onSubmit: (payload: AdjustStockLotPayload) => Promise<void>
}

export default function AdjustLotDialog({ open, lot, onClose, onSubmit }: Props) {
  const { t } = useTranslation('inventory')
  const [targetQuantity, setTargetQuantity] = useState('0')
  const [reason, setReason] = useState('')
  const [remark, setRemark] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    if (!open || !lot) return
    setTargetQuantity(String(lot.quantity_on_hand))
    setReason('')
    setRemark('')
    setError('')
  }, [open, lot])

  async function handleSubmit(event: React.FormEvent) {
    event.preventDefault()
    const nextQuantity = Number(targetQuantity)
    if (!Number.isFinite(nextQuantity) || nextQuantity < 0) {
      setError(t('adjustLot_errorInvalidQuantity'))
      return
    }

    setSubmitting(true)
    setError('')
    try {
      await onSubmit({
        target_quantity: nextQuantity,
        reason: reason.trim() || undefined,
        remark: remark.trim() || undefined,
      })
      onClose()
    } catch (err) {
      setError(err instanceof Error ? err.message : t('adjustLot_errorFailed'))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={(nextOpen) => !nextOpen && onClose()}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{t('adjustLot_title')}</DialogTitle>
          <DialogDescription>
            {lot
              ? t('adjustLot_descCurrent', { quantity: lot.quantity_on_hand, unit: lot.unit })
              : t('adjustLot_descEmpty')}
          </DialogDescription>
        </DialogHeader>

        <form className="space-y-4 py-2" onSubmit={handleSubmit}>
          <div className="space-y-1.5">
            <Label htmlFor="adjust-lot-quantity">{t('adjustLot_labelTargetQuantity')}</Label>
            <Input
              id="adjust-lot-quantity"
              type="number"
              min="0"
              step="any"
              value={targetQuantity}
              onChange={(event) => setTargetQuantity(event.target.value)}
            />
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="adjust-lot-reason">{t('adjustLot_labelReason')}</Label>
            <Input
              id="adjust-lot-reason"
              value={reason}
              onChange={(event) => setReason(event.target.value)}
              placeholder={t('adjustLot_placeholderReason')}
            />
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="adjust-lot-remark">{t('adjustLot_labelRemark')}</Label>
            <Input
              id="adjust-lot-remark"
              value={remark}
              onChange={(event) => setRemark(event.target.value)}
              placeholder={t('adjustLot_placeholderRemark')}
            />
          </div>

          {error && <p className="text-sm text-error">{error}</p>}

          <DialogFooter>
            <Button type="button" variant="outline" onClick={onClose} disabled={submitting}>
              {t('common:cancel')}
            </Button>
            <Button type="submit" disabled={submitting || !lot}>
              {submitting ? t('adjustLot_btnSubmitting') : t('adjustLot_btnSubmit')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
