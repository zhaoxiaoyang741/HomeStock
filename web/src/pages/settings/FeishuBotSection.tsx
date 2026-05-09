import { useEffect, useState, useCallback, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import { Bot, PowerOff, RefreshCw, ExternalLink, Loader2 } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Separator } from '@/components/ui/separator'
import { cn } from '@/lib/utils'
import { getFeishuAuthUrl, getFeishuStatus, disconnectFeishu } from '@/api/feishu'
import type { FeishuStatus } from '@/types/feishu'

export function FeishuBotSection() {
  const { t } = useTranslation('settings')
  const [status, setStatus] = useState<FeishuStatus | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [authLoading, setAuthLoading] = useState(false)
  const [disconnecting, setDisconnecting] = useState(false)
  const pollingRef = useRef<ReturnType<typeof setInterval> | null>(null)

  const fetchStatus = useCallback(async () => {
    try {
      const s = await getFeishuStatus()
      setStatus(s)
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

    // Poll every 10 seconds
    pollingRef.current = setInterval(() => {
      void fetchStatus()
    }, 10000)

    return () => {
      if (pollingRef.current) {
        clearInterval(pollingRef.current)
      }
    }
  }, [fetchStatus])

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
      setStatus((prev) => prev ? { ...prev, connected: false, bot_name: '' } : null)
    } catch (err) {
      setError(err instanceof Error ? err.message : t('feishuDisconnectFailed'))
    } finally {
      setDisconnecting(false)
    }
  }

  const isConfigured = status?.configured ?? false
  const isConnected = status?.connected ?? false

  return (
    <div className="space-y-6">
      <Card className="rounded-xl border-outline-variant/20 bg-surface-container-lowest shadow-sm">
        <CardHeader className="pb-4">
          <div className="flex items-center gap-3">
            <Bot className="w-5 h-5 text-primary" />
            <div>
              <CardTitle className="text-lg font-bold tracking-tight">{t('feishuBotTitle')}</CardTitle>
              <CardDescription>{t('feishuBotDescription')}</CardDescription>
            </div>
          </div>
        </CardHeader>
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

          {/* App info */}
          {isConfigured && (
            <div className="rounded-lg bg-surface-container px-3 py-2 text-sm text-on-surface-variant space-y-1">
              <div className="flex justify-between">
                <span>{t('feishuAppId')}</span>
                <span className="font-mono text-on-surface">{status?.app_id || '-'}</span>
              </div>
            </div>
          )}

          {/* Error message */}
          {error && (
            <div className="rounded-lg bg-error-container px-3 py-2 text-sm text-error">
              {error}
            </div>
          )}

          {/* Actions */}
          <div className="flex items-center gap-3 flex-wrap">
            {!isConnected ? (
              <Button
                variant="default"
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

          {/* Hint text */}
          {!isConfigured && (
            <p className="text-xs text-on-surface-variant">{t('feishuNotConfiguredHint')}</p>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
