import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Plus, Pencil, Trash2, Send, Loader2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { listEndpoints, createEndpoint, updateEndpoint, deleteEndpoint, testEndpoint, type EndpointConfig } from '@/api/outbound'

export function OutboundSection() {
  const { t } = useTranslation('settings')
  const [endpoints, setEndpoints] = useState<EndpointConfig[]>([])
  const [loading, setLoading] = useState(true)
  const [showForm, setShowForm] = useState(false)
  const [editing, setEditing] = useState<string | null>(null)
  const [name, setName] = useState('')
  const [url, setUrl] = useState('')
  const [enabled, setEnabled] = useState(true)
  const [saving, setSaving] = useState(false)
  const [testResult, setTestResult] = useState('')
  const [testing, setTesting] = useState(false)

  const fetchEndpoints = async () => {
    try {
      const data = await listEndpoints()
      setEndpoints(data)
    } catch {
      // handled
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void fetchEndpoints()
  }, [])

  const handleSave = async () => {
    setSaving(true)
    try {
      if (editing) {
        await updateEndpoint(editing, { url, enabled })
      } else {
        await createEndpoint({ name, url, enabled })
      }
      setShowForm(false)
      setEditing(null)
      setName('')
      setUrl('')
      setEnabled(true)
      await fetchEndpoints()
    } catch (e: unknown) {
      alert(e instanceof Error ? e.message : 'save failed')
    } finally {
      setSaving(false)
    }
  }

  const handleDelete = async (name: string) => {
    if (!confirm(t('common:confirmDelete'))) return
    try {
      await deleteEndpoint(name)
      await fetchEndpoints()
    } catch (e: unknown) {
      alert(e instanceof Error ? e.message : 'delete failed')
    }
  }

  const handleTest = async (testUrl: string) => {
    setTesting(true)
    setTestResult('')
    try {
      await testEndpoint({ url: testUrl })
      setTestResult('sent')
    } catch (e: unknown) {
      setTestResult(e instanceof Error ? e.message : 'test failed')
    } finally {
      setTesting(false)
    }
  }

  if (loading) {
    return <div className="flex justify-center py-8"><Loader2 className="h-6 w-6 animate-spin" /></div>
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-lg font-semibold text-on-surface">{t('outboundSectionTitle')}</h3>
          <p className="text-sm text-on-surface-variant">{t('outboundSectionDescription')}</p>
        </div>
        <Button onClick={() => { setShowForm(true); setEditing(null); setName(''); setUrl(''); setEnabled(true) }}>
          <Plus className="mr-2 h-4 w-4" />
          {t('outboundAdd')}
        </Button>
      </div>

      {showForm && (
        <Card>
          <CardHeader><CardTitle>{editing ? t('outboundEdit') : t('outboundAdd')}</CardTitle></CardHeader>
          <CardContent className="space-y-4">
            {!editing && (
              <div>
                <label className="text-sm font-medium">{t('outboundName')}</label>
                <Input value={name} onChange={e => setName(e.target.value)} placeholder="my-webhook" />
              </div>
            )}
            <div>
              <label className="text-sm font-medium">{t('outboundUrl')}</label>
              <Input value={url} onChange={e => setUrl(e.target.value)} placeholder="https://example.com/webhook" />
            </div>
            <label className="flex items-center gap-2 text-sm">
              <input type="checkbox" checked={enabled} onChange={e => setEnabled(e.target.checked)} className="rounded" />
              {t('outboundEnabled')}
            </label>
            <div className="flex gap-2">
              <Button onClick={() => void handleSave()} disabled={saving || !name || !url}>
                {saving ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : null}
                {t('common:save')}
              </Button>
              <Button variant="outline" onClick={() => { setShowForm(false); setEditing(null) }}>{t('common:cancel')}</Button>
            </div>
          </CardContent>
        </Card>
      )}

      <div className="space-y-3">
        {endpoints.map(ep => (
          <Card key={ep.name}>
            <CardContent className="flex items-center justify-between py-4">
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-2">
                  <span className="font-medium text-on-surface">{ep.name}</span>
                  <span className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ${
                    ep.enabled ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-500'
                  }`}>
                    {ep.enabled ? t('outboundEnabled') : t('outboundDisabled')}
                  </span>
                </div>
                <p className="mt-0.5 truncate text-sm text-on-surface-variant">{ep.url}</p>
              </div>
              <div className="flex items-center gap-1">
                <Button variant="ghost" size="icon" onClick={() => {
                  setEditing(ep.name)
                  setName(ep.name)
                  setUrl(ep.url)
                  setEnabled(ep.enabled)
                  setShowForm(true)
                }}>
                  <Pencil className="h-4 w-4" />
                </Button>
                <Button variant="ghost" size="icon" onClick={() => void handleTest(ep.url)} disabled={testing}>
                  <Send className={`h-4 w-4 ${testing ? 'animate-spin' : ''}`} />
                </Button>
                <Button variant="ghost" size="icon" onClick={() => void handleDelete(ep.name)}>
                  <Trash2 className="h-4 w-4 text-red-500" />
                </Button>
              </div>
            </CardContent>
          </Card>
        ))}
        {endpoints.length === 0 && (
          <p className="py-8 text-center text-sm text-on-surface-variant">{t('outboundEmpty')}</p>
        )}
      </div>

      {testResult && (
        <div className={`rounded-lg p-3 text-sm ${
          testResult === 'sent' ? 'bg-green-50 text-green-700' : 'bg-red-50 text-red-700'
        }`}>
          {testResult === 'sent' ? t('outboundTestSuccess') : `${t('outboundTestFailed')}: ${testResult}`}
        </div>
      )}
    </div>
  )
}
