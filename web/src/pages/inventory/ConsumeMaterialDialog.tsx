import { useEffect, useState } from 'react'
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
      setError('请先选择物料')
      return
    }
    if (!Number.isFinite(nextQuantity) || nextQuantity <= 0) {
      setError('消耗数量必须大于 0')
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
      setError(err instanceof Error ? err.message : '消耗失败')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={(nextOpen) => !nextOpen && onClose()}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>消耗库存</DialogTitle>
          <DialogDescription>
            {material ? `当前物料：${material.name}${material.spec ? ` / ${material.spec}` : ''}` : '请选择一个物料'}
          </DialogDescription>
        </DialogHeader>

        <form className="space-y-4 py-2" onSubmit={handleSubmit}>
          <div className="space-y-1.5">
            <Label htmlFor="consume-quantity">消耗数量</Label>
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
                当前总库存：{material.total_quantity} {material.default_unit}
              </p>
            )}
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="consume-reason">原因</Label>
            <Input
              id="consume-reason"
              value={reason}
              onChange={(event) => setReason(event.target.value)}
              placeholder="如：做饭、清理、损耗"
            />
          </div>

          {error && <p className="text-sm text-error">{error}</p>}

          <DialogFooter>
            <Button type="button" variant="outline" onClick={onClose} disabled={submitting}>
              取消
            </Button>
            <Button type="submit" disabled={submitting || !material}>
              {submitting ? '处理中…' : '确认消耗'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
