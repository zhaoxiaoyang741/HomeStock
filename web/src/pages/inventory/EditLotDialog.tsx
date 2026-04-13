import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import type { StockLot, UpdateStockLotPayload } from '@/types/stock'

interface Props {
  open: boolean
  lot: StockLot | null
  onClose: () => void
  onSubmit: (payload: UpdateStockLotPayload) => Promise<void>
}

function toDateInput(value: string | null): string {
  return value ? value.slice(0, 10) : ''
}

function toRFC3339(dateInput: string): string {
  return dateInput ? new Date(dateInput).toISOString() : ''
}

export default function EditLotDialog({ open, lot, onClose, onSubmit }: Props) {
  const { t } = useTranslation('inventory')
  const [location, setLocation] = useState('')
  const [notes, setNotes] = useState('')
  const [purchasedAt, setPurchasedAt] = useState('')
  const [expireAt, setExpireAt] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    if (!open || !lot) return
    setLocation(lot.location ?? '')
    setNotes(lot.notes ?? '')
    setPurchasedAt(toDateInput(lot.purchased_at))
    setExpireAt(toDateInput(lot.expire_at))
    setError('')
  }, [open, lot])

  async function handleSubmit(event: React.FormEvent) {
    event.preventDefault()
    setSubmitting(true)
    setError('')
    try {
      await onSubmit({
        location: location.trim(),
        notes: notes.trim(),
        purchased_at: purchasedAt ? toRFC3339(purchasedAt) : '',
        expire_at: expireAt ? toRFC3339(expireAt) : '',
      })
      onClose()
    } catch (err) {
      setError(err instanceof Error ? err.message : t('editLot_errorFailed'))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={(nextOpen) => !nextOpen && onClose()}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{t('editLot_title')}</DialogTitle>
        </DialogHeader>

        <form className="space-y-4 py-2" onSubmit={handleSubmit}>
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label htmlFor="edit-lot-location">{t('editLot_labelLocation')}</Label>
              <Input
                id="edit-lot-location"
                value={location}
                onChange={(event) => setLocation(event.target.value)}
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="edit-lot-purchased-at">{t('editLot_labelPurchasedAt')}</Label>
              <Input
                id="edit-lot-purchased-at"
                type="date"
                value={purchasedAt}
                onChange={(event) => setPurchasedAt(event.target.value)}
              />
            </div>
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label htmlFor="edit-lot-expire-at">{t('editLot_labelExpireAt')}</Label>
              <Input
                id="edit-lot-expire-at"
                type="date"
                value={expireAt}
                onChange={(event) => setExpireAt(event.target.value)}
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="edit-lot-notes">{t('editLot_labelNotes')}</Label>
              <Input
                id="edit-lot-notes"
                value={notes}
                onChange={(event) => setNotes(event.target.value)}
              />
            </div>
          </div>

          {error && <p className="text-sm text-error">{error}</p>}

          <DialogFooter>
            <Button type="button" variant="outline" onClick={onClose} disabled={submitting}>
              {t('common:cancel')}
            </Button>
            <Button type="submit" disabled={submitting}>
              {submitting ? t('editLot_btnSubmitting') : t('editLot_btnSubmit')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
