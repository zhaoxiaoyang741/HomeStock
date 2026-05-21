import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Checkbox } from '@/components/ui/checkbox'
import { Plus, Trash2 } from 'lucide-react'

interface ShoppingItem {
  id: string
  name: string
  createdAt: string
}

const STORAGE_KEY = 'homestock_shopping_list'

function loadItems(): ShoppingItem[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    return raw ? JSON.parse(raw) : []
  } catch {
    return []
  }
}

function saveItems(items: ShoppingItem[]) {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(items))
}

export default function ShoppingPage() {
  const { t } = useTranslation('nav')
  const { t: tc } = useTranslation('common')
  const [items, setItems] = useState<ShoppingItem[]>(loadItems)
  const [inputValue, setInputValue] = useState('')

  useEffect(() => {
    saveItems(items)
  }, [items])

  function addItem() {
    const name = inputValue.trim()
    if (!name) return
    setItems((prev) => [
      ...prev,
      { id: crypto.randomUUID(), name, createdAt: new Date().toISOString() },
    ])
    setInputValue('')
  }

  function handleKeyDown(event: React.KeyboardEvent<HTMLInputElement>) {
    if (event.key === 'Enter') addItem()
  }

  function toggleItem(id: string) {
    setItems((prev) => prev.filter((item) => item.id !== id))
  }

  return (
    <div className="flex flex-col h-full gap-6">
      <div className="flex items-center gap-3 flex-wrap">
        <div className="flex-1 min-w-0">
          <h1 className="text-2xl font-extrabold text-on-surface tracking-tight">{t('shopping')}</h1>
          <p className="text-sm text-on-surface-variant mt-0.5">添加需要购买的物品</p>
        </div>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>{t('shopping')}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex gap-2">
            <Input
              placeholder="输入商品名称..."
              value={inputValue}
              onChange={(e) => setInputValue(e.target.value)}
              onKeyDown={handleKeyDown}
              className="flex-1"
            />
            <Button onClick={addItem} disabled={!inputValue.trim()}>
              <Plus className="w-4 h-4 mr-1" />
              添加
            </Button>
          </div>

          {items.length === 0 ? (
            <p className="text-center text-on-surface-variant py-8">{tc('noData')}</p>
          ) : (
            <ul className="space-y-2">
              {items.map((item) => (
                <li
                  key={item.id}
                  className="flex items-center gap-3 px-3 py-2 rounded-lg hover:bg-surface-container transition-colors group"
                >
                  <Checkbox
                    checked={false}
                    onCheckedChange={() => toggleItem(item.id)}
                  />
                  <span className="flex-1 text-on-surface">{item.name}</span>
                  <button
                    onClick={() => toggleItem(item.id)}
                    className="p-1 rounded text-on-surface-variant hover:text-error hover:bg-error-container/50 opacity-0 group-hover:opacity-100 transition-all cursor-pointer"
                    title={tc('delete')}
                  >
                    <Trash2 className="w-4 h-4" />
                  </button>
                </li>
              ))}
            </ul>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
