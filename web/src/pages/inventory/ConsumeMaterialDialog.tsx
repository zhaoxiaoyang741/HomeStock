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
import type { ConsumeMaterialPayload, MaterialSummary } from '@/types/material'

interface Props {
  open: boolean
  material: MaterialSummary | null
  onClose: () => void
  onSubmit: (payload: ConsumeMaterialPayload) => Promise<void>
}

export default function ConsumeMaterialDialog({ open, material, onClose, onSubmit }: Props) {
  const { t } = useTranslation('inventory')
  const [quantity, setQuantity] = useState('1')
  const [reason, setReason] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    if (!open) return
    setQuantity('1')
    setReason('')
    setError('')
  }, [open, material])

  async function handleSubmit(event: React.FormEvent) {
    event.preventDefault()
    const nextQuantity = Number(quantity)
    if (!material) {
      setError(t('consume_errorNoMaterial'))
      return
    }
    if (!Number.isFinite(nextQuantity) || nextQuantity <= 0) {
      setError(t('consume_errorInvalidQuantity'))
      return
    }

    setSubmitting(true)
    setError('')
    try {
      await onSubmit({
        quantity: nextQuantity,
        reason: reason.trim() || undefined,
      })
      onClose()
    } catch (err) {
      setError(err instanceof Error ? err.message : t('consume_errorFailed'))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={(nextOpen) => !nextOpen && onClose()}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{t('consume_title')}</DialogTitle>
          <DialogDescription>
            {material
              ? t('consume_descCurrent', {
                  name: material.name,
                  spec: material.spec ? ` / ${material.spec}` : '',
                })
              : t('consume_descEmpty')}
          </DialogDescription>
        </DialogHeader>

        <form className="space-y-4 py-2" onSubmit={handleSubmit}>
          <div className="space-y-1.5">
            <Label htmlFor="consume-quantity">{t('consume_labelQuantity')}</Label>
            <Input
              id="consume-quantity"
              type="number"
              min="0"
              step="any"
              value={quantity}
              onChange={(event) => setQuantity(event.target.value)}
            />
            {material && (
              <p className="text-xs text-on-surface-variant">
                {t('consume_currentStock', {
                  quantity: material.total_quantity,
                  unit: material.default_unit,
                })}
              </p>
            )}
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="consume-reason">{t('common:reason')}</Label>
            <Input
              id="consume-reason"
              value={reason}
              onChange={(event) => setReason(event.target.value)}
              placeholder={t('consume_placeholderReason')}
            />
          </div>

          {error && <p className="text-sm text-error">{error}</p>}

          <DialogFooter>
            <Button type="button" variant="outline" onClick={onClose} disabled={submitting}>
              {t('common:cancel')}
            </Button>
            <Button type="submit" disabled={submitting || !material}>
              {submitting ? t('consume_btnSubmitting') : t('consume_btnSubmit')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
