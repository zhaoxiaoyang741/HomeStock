import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Tags, Loader2, Plus, Pencil, Trash2, X, Check } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Separator } from '@/components/ui/separator'
import { categoryApi } from '@/api/category'
import type { Category } from '@/types/category'

export function CategorySection() {
  const { t } = useTranslation('settings')
  const [categories, setCategories] = useState<Category[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  // Add state
  const [newName, setNewName] = useState('')
  const [adding, setAdding] = useState(false)

  // Edit state
  const [editingId, setEditingId] = useState<string | null>(null)
  const [editName, setEditName] = useState('')
  const [savingEdit, setSavingEdit] = useState<Record<string, boolean>>({})

  // Delete state
  const [deletingId, setDeletingId] = useState<string | null>(null)

  const fetchCategories = async () => {
    try {
      const data = await categoryApi.list()
      setCategories(data.categories)
      setError('')
    } catch (err) {
      setError(err instanceof Error ? err.message : t('categoryFetchFailed'))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void fetchCategories()
  }, [])

  async function handleAdd() {
    if (!newName.trim()) return
    setAdding(true)
    setError('')
    try {
      await categoryApi.create({ name: newName.trim() })
      setNewName('')
      await fetchCategories()
    } catch (err) {
      setError(err instanceof Error ? err.message : t('categoryAddFailed'))
    } finally {
      setAdding(false)
    }
  }

  function startEdit(cat: Category) {
    setEditingId(cat.id)
    setEditName(cat.name)
  }

  function cancelEdit() {
    setEditingId(null)
    setEditName('')
  }

  async function handleUpdate(id: string) {
    if (!editName.trim()) return
    setSavingEdit((prev) => ({ ...prev, [id]: true }))
    setError('')
    try {
      await categoryApi.update(id, { name: editName.trim() })
      setEditingId(null)
      setEditName('')
      await fetchCategories()
    } catch (err) {
      setError(err instanceof Error ? err.message : t('categoryUpdateFailed'))
    } finally {
      setSavingEdit((prev) => ({ ...prev, [id]: false }))
    }
  }

  async function handleDelete(id: string) {
    if (!window.confirm(t('categoryDeleteConfirm', { name: categories.find((c) => c.id === id)?.name ?? '' }))) {
      return
    }
    setDeletingId(id)
    setError('')
    try {
      await categoryApi.delete(id)
      await fetchCategories()
    } catch (err) {
      const msg = err instanceof Error ? err.message : String(err)
      if (msg.includes('in use') || msg.includes('关联')) {
        setError(t('categoryInUse'))
      } else {
        setError(msg || t('categoryDeleteFailed'))
      }
    } finally {
      setDeletingId(null)
    }
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center py-16">
        <Loader2 className="w-6 h-6 text-on-surface-variant animate-spin" />
      </div>
    )
  }

  return (
    <Card className="rounded-xl border-outline-variant/20 bg-surface-container-lowest shadow-sm">
      <CardHeader className="pb-4">
        <div className="flex items-center gap-3">
          <Tags className="w-5 h-5 text-primary" />
          <div>
            <CardTitle className="text-lg font-bold tracking-tight">{t('categorySectionTitle')}</CardTitle>
            <CardDescription>{t('categorySectionDescription')}</CardDescription>
          </div>
        </div>
      </CardHeader>
      <CardContent className="space-y-5">
        <Separator />

        {error && (
          <div className="rounded-lg bg-error-container px-3 py-2 text-sm text-error">
            {error}
          </div>
        )}

        {/* Add category form */}
        <div className="flex items-end gap-3">
          <div className="flex-1 space-y-1.5">
            <label className="text-sm font-medium text-on-surface" htmlFor="new-category-name">
              {t('categoryName')}
            </label>
            <Input
              id="new-category-name"
              value={newName}
              onChange={(e) => setNewName(e.target.value)}
              placeholder={t('categoryNamePlaceholder')}
              onKeyDown={(e) => { if (e.key === 'Enter') void handleAdd() }}
            />
          </div>
          <Button
            size="sm"
            onClick={() => void handleAdd()}
            disabled={adding || !newName.trim()}
          >
            {adding ? (
              <Loader2 className="mr-1 h-4 w-4 animate-spin" />
            ) : (
              <Plus className="mr-1 h-4 w-4" />
            )}
            {t('categoryAdd')}
          </Button>
        </div>

        {/* Category list */}
        {categories.length === 0 ? (
          <div className="text-center py-8 text-sm text-on-surface-variant">
            {t('categoryEmpty')}
          </div>
        ) : (
          <div className="space-y-2">
            {categories.map((cat) => (
              <div
                key={cat.id}
                className="flex items-center justify-between rounded-lg border border-outline-variant/20 bg-surface-container-low px-4 py-3"
              >
                {editingId === cat.id ? (
                  <div className="flex flex-1 items-center gap-2">
                    <Input
                      value={editName}
                      onChange={(e) => setEditName(e.target.value)}
                      className="h-8"
                      onKeyDown={(e) => { if (e.key === 'Enter') void handleUpdate(cat.id); if (e.key === 'Escape') cancelEdit() }}
                    />
                    <Button size="icon" variant="ghost" onClick={() => void handleUpdate(cat.id)} disabled={savingEdit[cat.id]}>
                      <Check className="h-4 w-4 text-primary" />
                    </Button>
                    <Button size="icon" variant="ghost" onClick={cancelEdit}>
                      <X className="h-4 w-4 text-on-surface-variant" />
                    </Button>
                  </div>
                ) : (
                  <>
                    <span className="text-sm font-medium text-on-surface">{cat.name}</span>
                    <div className="flex items-center gap-1">
                      <Button size="icon" variant="ghost" onClick={() => startEdit(cat)}>
                        <Pencil className="h-4 w-4 text-on-surface-variant" />
                      </Button>
                      <Button
                        size="icon"
                        variant="ghost"
                        onClick={() => void handleDelete(cat.id)}
                        disabled={deletingId === cat.id}
                      >
                        {deletingId === cat.id ? (
                          <Loader2 className="h-4 w-4 animate-spin text-error" />
                        ) : (
                          <Trash2 className="h-4 w-4 text-error" />
                        )}
                      </Button>
                    </div>
                  </>
                )}
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  )
}
