import { useEffect, useState, useCallback, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import { Bot, PowerOff, RefreshCw, ExternalLink, Loader2, Save, Pencil } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Checkbox } from '@/components/ui/checkbox'
import { Separator } from '@/components/ui/separator'
import { cn } from '@/lib/utils'
import { getFeishuAuthUrl, getFeishuStatus, disconnectFeishu, updateFeishuConfig, reconnectFeishu } from '@/api/feishu'
import type { FeishuStatus } from '@/types/feishu'

interface FeishuBotSectionProps {
  isActive: boolean
  onActivate: () => void
  onDeactivate: () => void
}

export function FeishuBotSection({ isActive, onActivate, onDeactivate }: FeishuBotSectionProps) {
  const { t } = useTranslation('settings')
  const [status, setStatus] = useState<FeishuStatus | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [authLoading, setAuthLoading] = useState(false)
  const [disconnecting, setDisconnecting] = useState(false)
  const [reconnecting, setReconnecting] = useState(false)

  // Config form state
  const [enabled, setEnabled] = useState(false)
  const [appId, setAppId] = useState('')
  const [appSecret, setAppSecret] = useState('')
  const [editingSecret, setEditingSecret] = useState(false)
  const [saving, setSaving] = useState(false)
  const [saveSuccess, setSaveSuccess] = useState('')

  const pollingRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const prevActiveRef = useRef(isActive)
  const initialSyncRef = useRef(false)

  const fetchStatus = useCallback(async () => {
    try {
      const s = await getFeishuStatus()
      setStatus(s)
      setEnabled(s.enabled)
      // Pre-populate app_id so the user can always see the current configured value
      if (s.app_id) {
        setAppId(s.app_id)
      }
      setError('')
    } catch (err) {
      if (!loading) {
        setError(err instanceof Error ? err.message : t('feishuStatusFailed'))
      }
    } finally {
      setLoading(false)
    }
  }, [loading, t])

  useEffect(() => {
    void fetchStatus()

    pollingRef.current = setInterval(() => {
      void fetchStatus()
    }, 10000)

    return () => {
      if (pollingRef.current) {
        clearInterval(pollingRef.current)
      }
    }
  }, [fetchStatus])

  // Sync initial enabled state to parent
  useEffect(() => {
    if (!loading && !initialSyncRef.current) {
      if (enabled) {
        onActivate()
      }
      initialSyncRef.current = true
    }
  }, [loading, enabled, onActivate])

  // Auto-disable when another channel takes over
  useEffect(() => {
    if (prevActiveRef.current && !isActive) {
      updateFeishuConfig({ enabled: false }).catch(() => {})
    }
    prevActiveRef.current = isActive
  }, [isActive])

  // Listen for OAuth callback result from redirected window
  useEffect(() => {
    const params = new URLSearchParams(window.location.search)
    const section = params.get('section')
    const authorized = params.get('authorized')
    const errMsg = params.get('error')

    if (section === 'channel') {
      if (authorized === '1') {
        void fetchStatus()
      }
      if (errMsg) {
        setError(decodeURIComponent(errMsg))
      }
      // Clean query params without reload
      const url = new URL(window.location.href)
      url.searchParams.delete('section')
      url.searchParams.delete('authorized')
      url.searchParams.delete('error')
      window.history.replaceState({}, '', url.toString())
    }
  }, [fetchStatus])

  async function handleSave() {
    setSaving(true)
    setError('')
    setSaveSuccess('')
    try {
      await updateFeishuConfig({
        enabled,
        app_id: appId || undefined,
        app_secret: appSecret || undefined,
      })
      setSaveSuccess(t('feishuSaveSuccess'))
      setAppSecret('')
      setEditingSecret(false)
      // Refresh status to get updated state (fetchStatus will re-populate appId)
      setLoading(true)
      await fetchStatus()
    } catch (err) {
      setError(err instanceof Error ? err.message : t('feishuSaveFailed'))
    } finally {
      setSaving(false)
    }
  }

  async function handleToggle(checked: boolean) {
    setEnabled(checked)
    try {
      await updateFeishuConfig({ enabled: checked })
      if (checked) {
        onActivate()
      } else {
        onDeactivate()
      }
    } catch (err) {
      // Revert local state on failure
      setEnabled(!checked)
      setError(err instanceof Error ? err.message : t('feishuSaveFailed'))
    }
  }

  async function handleAuthorize() {
    setAuthLoading(true)
    setError('')
    try {
      const { auth_url } = await getFeishuAuthUrl()
      window.open(auth_url, '_blank', 'noopener,noreferrer')
    } catch (err) {
      setError(err instanceof Error ? err.message : t('feishuAuthFailed'))
    } finally {
      setAuthLoading(false)
    }
  }

  async function handleDisconnect() {
    setDisconnecting(true)
    setError('')
    try {
      await disconnectFeishu()
      setStatus((prev) => prev ? { ...prev, connected: false, enabled: false, bot_name: '' } : null)
      setEnabled(false)
      onDeactivate()
    } catch (err) {
      setError(err instanceof Error ? err.message : t('feishuDisconnectFailed'))
    } finally {
      setDisconnecting(false)
    }
  }

  async function handleReconnect() {
    setReconnecting(true)
    setError('')
    try {
      await reconnectFeishu()
      await fetchStatus()
    } catch (err) {
      setError(err instanceof Error ? err.message : t('feishuReconnectFailed'))
    } finally {
      setReconnecting(false)
    }
  }

  const isConfigured = status?.configured ?? false
  const isConnected = status?.connected ?? false

  return (
    <div className="space-y-6">
      <Card className="rounded-xl border-outline-variant/20 bg-surface-container-lowest shadow-sm">
        <CardHeader className="pb-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <Bot className="w-5 h-5 text-primary" />
              <div>
                <CardTitle className="text-lg font-bold tracking-tight">{t('feishuBotTitle')}</CardTitle>
                <CardDescription>{t('feishuBotDescription')}</CardDescription>
              </div>
            </div>
            <Checkbox
              checked={enabled}
              onCheckedChange={(checked) => handleToggle(checked === true)}
            />
          </div>
        </CardHeader>

        {isActive && (
        <CardContent className="space-y-5">
          <Separator />

          {/* Connection status indicator */}
          <div className="flex items-center justify-between gap-3 flex-wrap">
            <div className="flex items-center gap-3">
              <div className={cn(
                'w-3 h-3 rounded-full',
                loading ? 'bg-on-surface-variant/30' :
                isConnected ? 'bg-primary' : 'bg-error'
              )} />
              <div className="space-y-0.5">
                <p className="text-sm font-medium text-on-surface">
                  {loading ? t('feishuStatusChecking') :
                   isConnected ? t('feishuConnected') : t('feishuDisconnected')}
                </p>
                {!loading && status?.bot_name && (
                  <p className="text-xs text-on-surface-variant">{status.bot_name}</p>
                )}
              </div>
            </div>
            <Badge variant={isConnected ? 'outline' : 'secondary'}>
              {isConnected ? t('feishuBadgeConnected') : t('feishuBadgeDisconnected')}
            </Badge>
          </div>

          {/* Error message */}
          {error && (
            <div className="rounded-lg bg-error-container px-3 py-2 text-sm text-error">
              {error}
            </div>
          )}

          {/* Success message */}
          {saveSuccess && (
            <div className="rounded-lg bg-primary/10 px-3 py-2 text-sm text-primary">
              {saveSuccess}
            </div>
          )}

          {/* Authorize / Reconnect / Disconnect / Refresh actions */}
          <div className="flex items-center gap-3 flex-wrap">
            {!isConnected ? (
              <>
                <Button
                  variant="default"
                  size="sm"
                  onClick={() => void handleReconnect()}
                  disabled={reconnecting || !isConfigured}
                >
                  {reconnecting ? (
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  ) : (
                    <RefreshCw className="mr-2 h-4 w-4" />
                  )}
                  {t('feishuReconnect')}
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => void handleAuthorize()}
                  disabled={authLoading || !isConfigured}
                >
                  {authLoading ? (
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  ) : (
                    <ExternalLink className="mr-2 h-4 w-4" />
                  )}
                  {t('feishuAuthorize')}
                </Button>
              </>
            ) : (
              <Button
                variant="destructive"
                size="sm"
                onClick={() => void handleDisconnect()}
                disabled={disconnecting}
              >
                {disconnecting ? (
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                ) : (
                  <PowerOff className="mr-2 h-4 w-4" />
                )}
                {t('feishuDisconnect')}
              </Button>
            )}
            <Button
              variant="outline"
              size="sm"
              onClick={() => { setLoading(true); void fetchStatus() }}
              disabled={loading}
            >
              <RefreshCw className={cn('mr-2 h-4 w-4', loading ? 'animate-spin' : '')} />
              {t('common:refresh')}
            </Button>
          </div>

          {(isConfigured || appId) && <Separator />}

          {/* Configuration form */}
          <div className="space-y-4">
            <div>
              <h4 className="text-sm font-medium text-on-surface">{t('feishuConfigTitle')}</h4>
              <p className="text-xs text-on-surface-variant mt-0.5">{t('feishuConfigHint')}</p>
            </div>

            {/* App ID */}
            <div className="space-y-1.5">
              <Label htmlFor="feishu-app-id">{t('feishuAppIdInput')}</Label>
              <Input
                id="feishu-app-id"
                placeholder={t('feishuPlaceholderKeep')}
                value={appId}
                onChange={(e) => setAppId(e.target.value)}
              />
            </div>

            {/* App Secret */}
            <div className="space-y-1.5">
              <Label htmlFor="feishu-app-secret">{t('feishuAppSecretInput')}</Label>
              {editingSecret ? (
                <>
                  <Input
                    id="feishu-app-secret"
                    type="password"
                    placeholder={isConfigured ? '········' : t('feishuPlaceholderKeep')}
                    value={appSecret}
                    onChange={(e) => setAppSecret(e.target.value)}
                  />
                  <p className="text-xs text-on-surface-variant">{t('feishuAppSecretHint')}</p>
                </>
              ) : (
                <div className="flex items-center gap-2">
                  <span className="text-sm text-on-surface-variant">••••••••</span>
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={() => setEditingSecret(true)}
                  >
                    <Pencil className="w-3.5 h-3.5 mr-1" />
                    {t('feishuChangeSecret')}
                  </Button>
                </div>
              )}
            </div>

            {/* Save button */}
            <Button
              size="sm"
              onClick={() => void handleSave()}
              disabled={saving || (enabled === status?.enabled && appId === status?.app_id && !appSecret)}
            >
              {saving ? (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              ) : (
                <Save className="mr-2 h-4 w-4" />
              )}
              {saving ? t('feishuSaving') : t('common:save')}
            </Button>
          </div>

          {/* Hint text */}
          {!isConfigured && (
            <p className="text-xs text-on-surface-variant">{t('feishuNotConfiguredHint')}</p>
          )}
        </CardContent>
        )}
      </Card>
    </div>
  )
}
