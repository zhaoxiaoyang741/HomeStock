import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Key, Plus, Trash2, Loader2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { listAPIKeys, addAPIKey, deleteAPIKey } from '@/api/apiKeys'

export function ApiKeySection() {
  const { t } = useTranslation('settings')
  const [keys, setKeys] = useState<string[]>([])
  const [loading, setLoading] = useState(true)
  const [newKey, setNewKey] = useState('')
  const [adding, setAdding] = useState(false)
  const [showAdd, setShowAdd] = useState(false)

  const fetchKeys = async () => {
    try {
      const data = await listAPIKeys()
      setKeys(data)
    } catch {
      // handled
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void fetchKeys()
  }, [])

  const handleAdd = async () => {
    if (!newKey.trim()) return
    setAdding(true)
    try {
      await addAPIKey(newKey.trim())
      setNewKey('')
      setShowAdd(false)
      await fetchKeys()
    } catch (e: unknown) {
      alert(e instanceof Error ? e.message : 'add failed')
    } finally {
      setAdding(false)
    }
  }

  const handleDelete = async (key: string) => {
    if (!confirm(t('common:confirmDelete'))) return
    try {
      await deleteAPIKey(key)
      await fetchKeys()
    } catch (e: unknown) {
      alert(e instanceof Error ? e.message : 'delete failed')
    }
  }

  if (loading) {
    return <div className="flex justify-center py-8"><Loader2 className="h-6 w-6 animate-spin" /></div>
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-lg font-semibold text-on-surface">{t('apiKeySectionTitle')}</h3>
          <p className="text-sm text-on-surface-variant">{t('apiKeySectionDescription')}</p>
        </div>
        <Button onClick={() => setShowAdd(true)}>
          <Plus className="mr-2 h-4 w-4" />
          {t('apiKeyAdd')}
        </Button>
      </div>

      {showAdd && (
        <Card>
          <CardHeader><CardTitle>{t('apiKeyAdd')}</CardTitle></CardHeader>
          <CardContent className="space-y-4">
            <div>
              <label className="text-sm font-medium">{t('apiKeyLabel')}</label>
              <Input
                value={newKey}
                onChange={e => setNewKey(e.target.value)}
                placeholder={t('apiKeyPlaceholder')}
              />
            </div>
            <div className="flex gap-2">
              <Button onClick={() => void handleAdd()} disabled={adding || !newKey.trim()}>
                {adding ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : null}
                {t('common:save')}
              </Button>
              <Button variant="outline" onClick={() => { setShowAdd(false); setNewKey('') }}>
                {t('common:cancel')}
              </Button>
            </div>
          </CardContent>
        </Card>
      )}

      <div className="space-y-3">
        {keys.map(key => (
          <Card key={key}>
            <CardContent className="flex items-center justify-between py-4">
              <div className="flex items-center gap-3 min-w-0 flex-1">
                <Key className="h-4 w-4 shrink-0 text-on-surface-variant" />
                <span className="font-mono text-sm text-on-surface truncate">{key}</span>
              </div>
              <Button variant="ghost" size="icon" onClick={() => void handleDelete(key)}>
                <Trash2 className="h-4 w-4 text-red-500" />
              </Button>
            </CardContent>
          </Card>
        ))}
        {keys.length === 0 && (
          <p className="py-8 text-center text-sm text-on-surface-variant">{t('apiKeyEmpty')}</p>
        )}
      </div>
    </div>
  )
}
