import { useEffect, useState } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import type { Item, CreateItemPayload, UpdateItemPayload } from '@/types/item'
import type { Category } from '@/types/category'

interface Props {
  open: boolean
  item: Item | null
  categories: Category[]
  onClose: () => void
  onSubmit: (payload: CreateItemPayload | UpdateItemPayload) => Promise<void>
}

interface FormState {
  name: string
  category_id: string
  quantity: string
  unit: string
  location: string
  expire_at: string
  purchased_at: string
  notes: string
}

function toDateInput(iso: string | null | undefined): string {
  if (!iso) return ''
  return iso.slice(0, 10)
}

function toRFC3339(dateInput: string): string {
  return dateInput ? new Date(dateInput).toISOString() : ''
}

const EMPTY_FORM: FormState = {
  name: '',
  category_id: '',
  quantity: '1',
  unit: '个',
  location: '',
  expire_at: '',
  purchased_at: toDateInput(new Date().toISOString()),
  notes: '',
}

export default function ItemFormDialog({ open, item, categories, onClose, onSubmit }: Props) {
  const [form, setForm] = useState<FormState>(EMPTY_FORM)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    if (open) {
      if (item) {
        setForm({
          name: item.name,
          category_id: item.category_id ?? '',
          quantity: String(item.quantity),
          unit: item.unit,
          location: item.location,
          expire_at: toDateInput(item.expire_at),
          purchased_at: toDateInput(item.purchased_at),
          notes: item.notes,
        })
      } else {
        const defaultCategory = categories.find((c) => c.name === '默认分类')
        setForm({ ...EMPTY_FORM, category_id: defaultCategory?.id ?? '' })
      }
      setError('')
    }
  }, [open, item, categories])

  function set(key: keyof FormState, value: string) {
    setForm((f) => ({ ...f, [key]: value }))
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!form.name.trim()) {
      setError('物料名称不能为空')
      return
    }
    const payload: CreateItemPayload = {
      name: form.name.trim(),
      category_id: form.category_id || undefined,
      quantity: parseFloat(form.quantity) || 1,
      unit: form.unit.trim() || '个',
      location: form.location.trim(),
      expire_at: form.expire_at ? toRFC3339(form.expire_at) : undefined,
      purchased_at: form.purchased_at ? toRFC3339(form.purchased_at) : undefined,
      notes: form.notes.trim(),
    }
    setSubmitting(true)
    setError('')
    try {
      await onSubmit(payload)
      onClose()
    } catch (err) {
      setError(err instanceof Error ? err.message : '操作失败')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={(v) => !v && onClose()}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{item ? '编辑物料' : '新增物料'}</DialogTitle>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-4 py-2">
          {/* 名称 */}
          <div className="space-y-1.5">
            <Label htmlFor="item-name">
              物料名称 <span className="text-error">*</span>
            </Label>
            <Input
              id="item-name"
              value={form.name}
              onChange={(e) => set('name', e.target.value)}
              placeholder="请输入物料名称"
            />
          </div>

          {/* 分类 + 单位 */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label>分类</Label>
              <Select value={form.category_id} onValueChange={(v) => set('category_id', v === '__none__' ? '' : v)}>
                <SelectTrigger>
                  <SelectValue placeholder="选择分类" />
                </SelectTrigger>
                <SelectContent>
                  {categories.map((c) => (
                    <SelectItem key={c.id} value={c.id}>
                      {c.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-1.5">
              {/* TODO 通过后端维护name->单位的映射表，这里进行推测 */}
              <Label htmlFor="item-unit">单位</Label>
              <Input
                id="item-unit"
                value={form.unit}
                onChange={(e) => set('unit', e.target.value)}
                placeholder="个"
              />
            </div>
          </div>

          {/* 数量 + 存放位置 */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label htmlFor="item-quantity">数量</Label>
              <Input
                id="item-quantity"
                type="number"
                min="0"
                step="any"
                value={form.quantity}
                onChange={(e) => set('quantity', e.target.value)}
              />
            </div>

            <div className="space-y-1.5">
              <Label htmlFor="item-location">存放位置</Label>
              <Input
                id="item-location"
                value={form.location}
                onChange={(e) => set('location', e.target.value)}
                placeholder="如：厨房、卫生间"
              />
            </div>
          </div>

          {/* 购买日期 + 过期日期 */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label htmlFor="item-purchased-at">购买日期</Label>
              <Input
                id="item-purchased-at"
                type="date"
                value={form.purchased_at}
                onChange={(e) => set('purchased_at', e.target.value)}
              />
            </div>

            <div className="space-y-1.5">
              <Label htmlFor="item-expire-at">过期日期</Label>
              <Input
                id="item-expire-at"
                type="date"
                value={form.expire_at}
                onChange={(e) => set('expire_at', e.target.value)}
              />
            </div>
          </div>

          {/* 备注 */}
          <div className="space-y-1.5">
            <Label htmlFor="item-notes">备注</Label>
            <textarea
              id="item-notes"
              value={form.notes}
              onChange={(e) => set('notes', e.target.value)}
              placeholder="可选备注..."
              rows={3}
              className="w-full rounded-md border border-outline-variant bg-surface px-3 py-2 text-sm text-on-surface placeholder:text-on-surface-variant focus:outline-none focus:ring-2 focus:ring-primary/50 resize-none"
            />
          </div>

          {error && <p className="text-sm text-error">{error}</p>}

          <DialogFooter>
            <Button type="button" variant="outline" onClick={onClose} disabled={submitting}>
              取消
            </Button>
            <Button type="submit" disabled={submitting}>
              {submitting ? '保存中…' : '保存'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
