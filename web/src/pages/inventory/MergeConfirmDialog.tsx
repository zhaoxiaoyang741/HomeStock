import { useState } from 'react'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  DialogDescription,
} from '@/components/ui/dialog'
import type { Item, CreateItemPayload } from '@/types/item'

interface Props {
  open: boolean
  existingItem: Item
  incomingItem: CreateItemPayload
  onClose: () => void
  onMerge: () => Promise<void>
  onCreate: () => Promise<void>
}

export default function MergeConfirmDialog({
  open,
  existingItem,
  incomingItem,
  onClose,
  onMerge,
  onCreate,
}: Props) {
  const [acting, setActing] = useState(false)

  const incomingQty = incomingItem.quantity ?? 1
  const mergedQty = existingItem.quantity + incomingQty
  const unit = existingItem.unit

  async function handleMerge() {
    setActing(true)
    try {
      await onMerge()
    } finally {
      setActing(false)
    }
  }

  async function handleCreate() {
    setActing(true)
    try {
      await onCreate()
    } finally {
      setActing(false)
    }
  }

  const existingLabel = existingItem.spec
    ? `${existingItem.name}（${existingItem.spec}）`
    : existingItem.name

  return (
    <Dialog open={open} onOpenChange={(v) => !v && onClose()}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>发现相似物料</DialogTitle>
          <DialogDescription>
            库存中已有相同名称{existingItem.spec ? '和规格' : ''}的物料，请选择操作方式。
          </DialogDescription>
        </DialogHeader>

        <div className="rounded-lg border border-outline-variant bg-surface-container p-4 space-y-3 text-sm">
          <div className="space-y-1">
            <p className="text-on-surface-variant text-xs font-medium uppercase tracking-wide">已有物料</p>
            <p className="text-on-surface font-medium">{existingLabel}</p>
            <p className="text-on-surface-variant">
              当前库存：{existingItem.quantity} {unit}
              {existingItem.location && `　存放：${existingItem.location}`}
            </p>
          </div>

          <div className="border-t border-outline-variant pt-3 space-y-1">
            <p className="text-on-surface-variant text-xs font-medium uppercase tracking-wide">本次新增</p>
            <p className="text-on-surface-variant">+{incomingQty} {unit}</p>
          </div>

          <div className="border-t border-outline-variant pt-3">
            <p className="text-on-surface-variant text-xs font-medium uppercase tracking-wide mb-1">合并后总量</p>
            <p className="text-primary font-bold text-base">{mergedQty} {unit}</p>
          </div>
        </div>

        <DialogFooter className="gap-2 sm:gap-0">
          <Button variant="outline" onClick={onClose} disabled={acting}>
            取消
          </Button>
          <Button variant="outline" onClick={handleCreate} disabled={acting}>
            新建条目
          </Button>
          <Button onClick={handleMerge} disabled={acting}>
            {acting ? '处理中…' : `合并（+${incomingQty} ${unit}）`}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
