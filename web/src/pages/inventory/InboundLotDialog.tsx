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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import type { Category } from '@/types/category'
import type { InboundStockLotPayload } from '@/types/stock'

interface Props {
  open: boolean
  categories: Category[]
  initialName?: string
  initialSpec?: string
  onClose: () => void
  onSubmit: (payload: InboundStockLotPayload) => Promise<void>
}

interface FormState {
  name: string
  spec: string
  category_id: string
  quantity: string
  unit: string
  location: string
  purchased_at: string
  expire_at: string
  notes: string
}

function toRFC3339(dateInput: string): string {
  return dateInput ? new Date(dateInput).toISOString() : ''
}

export default function InboundLotDialog({
  open,
  categories,
  initialName,
  initialSpec,
  onClose,
  onSubmit,
}: Props) {
  const { t, i18n } = useTranslation('inventory')
  const [form, setForm] = useState<FormState>(() => ({
    name: '',
    spec: '',
    category_id: '',
    quantity: '1',
    unit: i18n.language === 'en' ? 'pcs' : '件',
    location: '',
    purchased_at: new Date().toISOString().slice(0, 10),
    expire_at: '',
    notes: '',
  }))
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    if (!open) return
    const defaultCategory = categories.find((category) => category.name === '默认分类')
    setForm({
      name: initialName ?? '',
      spec: initialSpec ?? '',
      category_id: defaultCategory?.id ?? '',
      quantity: '1',
      unit: i18n.language === 'en' ? 'pcs' : '件',
      location: '',
      purchased_at: new Date().toISOString().slice(0, 10),
      expire_at: '',
      notes: '',
    })
    setError('')
  }, [open, categories, initialName, initialSpec, i18n.language])

  function setValue<K extends keyof FormState>(key: K, value: FormState[K]) {
    setForm((current) => ({ ...current, [key]: value }))
  }

  async function handleSubmit(event: React.FormEvent) {
    event.preventDefault()
    if (!form.name.trim()) {
      setError(t('inbound_errorNameRequired'))
      return
    }

    setSubmitting(true)
    setError('')
    try {
      await onSubmit({
        name: form.name.trim(),
        spec: form.spec.trim() || undefined,
        category_id: form.category_id || undefined,
        quantity: Number(form.quantity) || 1,
        unit: form.unit.trim() || (i18n.language === 'en' ? 'pcs' : '件'),
        location: form.location.trim() || undefined,
        purchased_at: form.purchased_at ? toRFC3339(form.purchased_at) : undefined,
        expire_at: form.expire_at ? toRFC3339(form.expire_at) : undefined,
        notes: form.notes.trim() || undefined,
      })
      onClose()
    } catch (err) {
      setError(err instanceof Error ? err.message : t('inbound_errorFailed'))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={(nextOpen) => !nextOpen && onClose()}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{t('inbound_title')}</DialogTitle>
        </DialogHeader>

        <form className="space-y-4 py-2" onSubmit={handleSubmit}>
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label htmlFor="inbound-name">{t('inbound_labelName')}</Label>
              <Input
                id="inbound-name"
                value={form.name}
                onChange={(event) => setValue('name', event.target.value)}
                placeholder={t('inbound_placeholderName')}
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="inbound-spec">{t('inbound_labelSpec')}</Label>
              <Input
                id="inbound-spec"
                value={form.spec}
                onChange={(event) => setValue('spec', event.target.value)}
                placeholder={t('inbound_placeholderSpec')}
              />
            </div>
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label>{t('inbound_labelCategory')}</Label>
              <Select value={form.category_id} onValueChange={(value) => setValue('category_id', value)}>
                <SelectTrigger>
                  <SelectValue placeholder={t('inbound_placeholderCategory')} />
                </SelectTrigger>
                <SelectContent>
                  {categories.map((category) => (
                    <SelectItem key={category.id} value={category.id}>
                      {category.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="inbound-location">{t('inbound_labelLocation')}</Label>
              <Input
                id="inbound-location"
                value={form.location}
                onChange={(event) => setValue('location', event.target.value)}
                placeholder={t('inbound_placeholderLocation')}
              />
            </div>
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label htmlFor="inbound-quantity">{t('inbound_labelQuantity')}</Label>
              <Input
                id="inbound-quantity"
                type="number"
                min="0"
                step="any"
                value={form.quantity}
                onChange={(event) => setValue('quantity', event.target.value)}
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="inbound-unit">{t('inbound_labelUnit')}</Label>
              <Input
                id="inbound-unit"
                value={form.unit}
                onChange={(event) => setValue('unit', event.target.value)}
                placeholder={t('inbound_placeholderUnit')}
              />
            </div>
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label htmlFor="inbound-purchased-at">{t('inbound_labelPurchasedAt')}</Label>
              <Input
                id="inbound-purchased-at"
                type="date"
                value={form.purchased_at}
                onChange={(event) => setValue('purchased_at', event.target.value)}
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="inbound-expire-at">{t('inbound_labelExpireAt')}</Label>
              <Input
                id="inbound-expire-at"
                type="date"
                value={form.expire_at}
                onChange={(event) => setValue('expire_at', event.target.value)}
              />
            </div>
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="inbound-notes">{t('inbound_labelNotes')}</Label>
            <textarea
              id="inbound-notes"
              rows={3}
              value={form.notes}
              onChange={(event) => setValue('notes', event.target.value)}
              className="w-full rounded-lg border border-outline-variant bg-surface px-3 py-2 text-sm text-on-surface placeholder:text-on-surface-variant focus:outline-none focus:ring-2 focus:ring-primary/40 resize-none"
              placeholder={t('inbound_placeholderNotes')}
            />
          </div>

          {error && <p className="text-sm text-error">{error}</p>}

          <DialogFooter>
            <Button type="button" variant="outline" onClick={onClose} disabled={submitting}>
              {t('common:cancel')}
            </Button>
            <Button type="submit" disabled={submitting}>
              {submitting ? t('inbound_btnSubmitting') : t('inbound_btnSubmit')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
