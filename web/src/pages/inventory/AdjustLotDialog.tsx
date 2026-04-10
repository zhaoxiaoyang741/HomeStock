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
import type { AdjustStockLotPayload, StockLot } from '@/types/stock'

interface Props {
  open: boolean
  lot: StockLot | null
  onClose: () => void
  onSubmit: (payload: AdjustStockLotPayload) => Promise<void>
}

export default function AdjustLotDialog({ open, lot, onClose, onSubmit }: Props) {
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
      setError('目标库存不能小于 0')
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
      setError(err instanceof Error ? err.message : '调整库存失败')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={(nextOpen) => !nextOpen && onClose()}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>调整批次库存</DialogTitle>
          <DialogDescription>
            {lot ? `当前批次库存：${lot.quantity_on_hand} ${lot.unit}` : '请选择一个批次'}
          </DialogDescription>
        </DialogHeader>

        <form className="space-y-4 py-2" onSubmit={handleSubmit}>
          <div className="space-y-1.5">
            <Label htmlFor="adjust-lot-quantity">目标库存</Label>
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
            <Label htmlFor="adjust-lot-reason">原因</Label>
            <Input
              id="adjust-lot-reason"
              value={reason}
              onChange={(event) => setReason(event.target.value)}
              placeholder="如：盘点、修正、损耗"
            />
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="adjust-lot-remark">备注</Label>
            <Input
              id="adjust-lot-remark"
              value={remark}
              onChange={(event) => setRemark(event.target.value)}
              placeholder="补充说明"
            />
          </div>

          {error && <p className="text-sm text-error">{error}</p>}

          <DialogFooter>
            <Button type="button" variant="outline" onClick={onClose} disabled={submitting}>
              取消
            </Button>
            <Button type="submit" disabled={submitting || !lot}>
              {submitting ? '调整中…' : '确认调整'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
