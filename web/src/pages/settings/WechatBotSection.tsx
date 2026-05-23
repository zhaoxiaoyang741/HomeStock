import { useEffect, useState, useCallback, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import { MessageSquare, PowerOff, RefreshCw, Loader2, QrCode, Check, X, AlertTriangle } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Separator } from '@/components/ui/separator'
import { cn } from '@/lib/utils'
import { getWechatStatus, disconnectWechat, reconnectWechat, startWechatQRFlow, pollWechatQRFlow, updateWechatConfig } from '@/api/wechat'
import type { WechatStatus } from '@/types/wechat'

type BindingState = 'idle' | 'loading' | 'waiting' | 'scaned' | 'confirmed' | 'expired' | 'error'

interface WechatBotSectionProps {
  isActive: boolean
  onActivate: () => void
  onDeactivate: () => void
}

export function WechatBotSection({ isActive, onActivate, onDeactivate }: WechatBotSectionProps) {
  const { t } = useTranslation('settings')
  const [status, setStatus] = useState<WechatStatus | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [connecting, setConnecting] = useState(false)
  const [disconnecting, setDisconnecting] = useState(false)

  // QR binding state
  const [bindState, setBindState] = useState<BindingState>('idle')
  const [qrDataURI, setQrDataURI] = useState<string | null>(null)
  const [accountID, setAccountID] = useState<string | null>(null)
  const [bindError, setBindError] = useState('')

  const [enabled, setEnabled] = useState(false)
  const pollingRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const initialRef = useRef(true)
  const pollTimerRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const pollGenerationRef = useRef(0)
  const prevActiveRef = useRef(isActive)
  const initialSyncRef = useRef(false)

  // Stop QR polling
  const stopQrPolling = useCallback(() => {
    pollGenerationRef.current += 1
    if (pollTimerRef.current !== null) {
      clearInterval(pollTimerRef.current)
      pollTimerRef.current = null
    }
  }, [])

  useEffect(() => () => stopQrPolling(), [stopQrPolling])

  const refreshStatus = useCallback(() => {
    setLoading(true)
    getWechatStatus()
      .then((s) => {
        setStatus(s)
        setError('')
      })
      .catch((err) => {
        setError(err instanceof Error ? err.message : t('wechatStatusFailed'))
      })
      .finally(() => {
        setLoading(false)
      })
  }, [t])

  // Start QR polling
  const startQrPolling = useCallback((flowID: string) => {
    stopQrPolling()
    const generation = pollGenerationRef.current
    let inFlight = false
    pollTimerRef.current = setInterval(async () => {
      if (inFlight) return
      inFlight = true
      try {
        const resp = await pollWechatQRFlow(flowID)
        if (generation !== pollGenerationRef.current) return
        if (resp.status === 'scaned') {
          setBindState('scaned')
        } else if (resp.status === 'confirmed') {
          stopQrPolling()
          setAccountID(resp.account_id ?? null)
          setBindState('confirmed')
          setEnabled(true)
          onActivate()
          // Refresh status after binding
          setTimeout(() => refreshStatus(), 500)
        } else if (resp.status === 'expired') {
          stopQrPolling()
          setBindState('expired')
        } else if (resp.status === 'error') {
          stopQrPolling()
          setBindState('error')
          setBindError(resp.error ?? t('wechatBindError'))
        }
      } catch {
        // transient network error — keep polling
      } finally {
        inFlight = false
      }
    }, 2000)
  }, [stopQrPolling, t, refreshStatus, onActivate])

  // Status polling
  useEffect(() => {
    const poll = () => {
      getWechatStatus()
        .then((s) => {
          setStatus(s)
          setEnabled(s.enabled)
          setError('')
          // If token expired, don't show confirmed state
          if (s.token_expired) {
            setBindState('idle')
          } else if (s.has_token && s.account_id) {
            setBindState('confirmed')
            setAccountID(s.account_id)
          }
        })
        .catch((err) => {
          if (!initialRef.current) {
            setError(err instanceof Error ? err.message : t('wechatStatusFailed'))
          }
        })
        .finally(() => {
          initialRef.current = false
          setLoading(false)
        })
    }
    poll()
    pollingRef.current = setInterval(poll, 10000)
    return () => {
      if (pollingRef.current) clearInterval(pollingRef.current)
    }
  }, [t])

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
      updateWechatConfig({ enabled: false }).catch(() => {})
      setEnabled(false)
    }
    prevActiveRef.current = isActive
  }, [isActive])

  async function handleToggle(checked: boolean) {
    setEnabled(checked)
    try {
      await updateWechatConfig({ enabled: checked })
      if (checked) {
        onActivate()
      } else {
        onDeactivate()
      }
    } catch {
      setEnabled(!checked)
    }
  }

  async function handleConnect() {
    setConnecting(true)
    setError('')
    try {
      await reconnectWechat()
      setTimeout(() => refreshStatus(), 1000)
    } catch (err) {
      setError(err instanceof Error ? err.message : t('wechatConnectFailed'))
    } finally {
      setConnecting(false)
    }
  }

  async function handleDisconnect() {
    setDisconnecting(true)
    setError('')
    try {
      await disconnectWechat()
      setStatus((prev) => prev ? { ...prev, connected: false, account_id: '' } : null)
      setEnabled(false)
      onDeactivate()
    } catch (err) {
      setError(err instanceof Error ? err.message : t('wechatDisconnectFailed'))
    } finally {
      setDisconnecting(false)
    }
  }

  async function handleBind() {
    setBindState('loading')
    setBindError('')
    setQrDataURI(null)
    stopQrPolling()
    try {
      const resp = await startWechatQRFlow()
      setQrDataURI(resp.qr_data_uri ?? null)
      setBindState('waiting')
      startQrPolling(resp.flow_id)
    } catch (e) {
      setBindState('error')
      setBindError(e instanceof Error ? e.message : t('wechatBindError'))
    }
  }

  function handleRebind() {
    stopQrPolling()
    setQrDataURI(null)
    setAccountID(null)
    setBindError('')
    // Directly start a new QR code binding flow instead of just resetting UI
    handleBind()
  }

  const isConnected = status?.connected ?? false
  const isEnabled = status?.enabled ?? false
  const nickname = status?.account_id ?? ''
  const isBound = status?.has_token ?? false
  const tokenExpired = status?.token_expired ?? false

  // Render QR binding section
  const renderBindSection = () => {
    if (bindState === 'idle') {
      if (isBound && tokenExpired) {
        return (
          <div className="flex flex-col items-center gap-4 py-4">
            <div className="flex items-center gap-2 rounded-full bg-red-500/10 px-4 py-2 text-sm font-medium text-red-600 dark:text-red-400">
              <AlertTriangle className="h-4 w-4" />
              {t('wechatTokenExpired')}
            </div>
            {accountID && (
              <p className="font-mono text-xs text-on-surface-variant">{accountID}</p>
            )}
            <Button onClick={() => void handleBind()} className="gap-2" variant="default">
              <QrCode className="h-4 w-4" />
              {t('wechatRebindAction')}
            </Button>
          </div>
        )
      }
      if (isBound) {
        return (
          <div className="flex flex-col items-center gap-3 py-4">
            <div className="flex items-center gap-2 rounded-full bg-emerald-500/10 px-4 py-2 text-sm font-medium text-emerald-600 dark:text-emerald-400">
              <Check className="h-4 w-4" />
              {t('wechatBound')}
            </div>
            {accountID && (
              <p className="font-mono text-xs text-on-surface-variant">{accountID}</p>
            )}
            <Button variant="outline" size="sm" onClick={handleRebind} className="mt-1 gap-2">
              <RefreshCw className="h-3.5 w-3.5" />
              {t('wechatRebind')}
            </Button>
          </div>
        )
      }
      return (
        <div className="flex flex-col items-center gap-4 py-4">
          <p className="text-sm text-on-surface-variant">{t('wechatNotBound')}</p>
          <Button onClick={() => void handleBind()} className="gap-2">
            <QrCode className="h-4 w-4" />
            {t('wechatBind')}
          </Button>
        </div>
      )
    }

    if (bindState === 'loading') {
      return (
        <div className="flex flex-col items-center gap-3 py-6">
          <Loader2 className="h-8 w-8 animate-spin text-on-surface-variant" />
          <p className="text-sm text-on-surface-variant">{t('wechatGeneratingQR')}</p>
        </div>
      )
    }

    if (bindState === 'waiting' || bindState === 'scaned') {
      return (
        <div className="flex flex-col items-center gap-4 py-4">
          {qrDataURI ? (
            <img
              src={qrDataURI}
              alt="WeChat QR Code"
              className="h-48 w-48 rounded-xl border border-outline-variant/60 bg-white p-2 shadow-sm"
            />
          ) : (
            <div className="flex h-48 w-48 items-center justify-center rounded-xl border border-outline-variant/60 bg-surface-container">
              <Loader2 className="h-8 w-8 animate-spin text-on-surface-variant" />
            </div>
          )}
          {bindState === 'scaned' ? (
            <div className="flex items-center gap-2 rounded-full bg-amber-500/10 px-4 py-2 text-sm font-medium text-amber-600 dark:text-amber-400">
              <Loader2 className="h-3.5 w-3.5 animate-spin" />
              {t('wechatScanned')}
            </div>
          ) : (
            <p className="text-sm text-on-surface-variant">{t('wechatScanHint')}</p>
          )}
          <Button variant="ghost" size="sm" onClick={handleRebind} className="text-on-surface-variant">
            <RefreshCw className="mr-1 h-3.5 w-3.5" />
            {t('common:refresh')}
          </Button>
        </div>
      )
    }

    if (bindState === 'confirmed') {
      return (
        <div className="flex flex-col items-center gap-3 py-4">
          <div className="flex h-14 w-14 items-center justify-center rounded-full bg-emerald-500/10">
            <Check className="h-7 w-7 text-emerald-600 dark:text-emerald-400" />
          </div>
          <p className="text-sm font-medium text-emerald-600 dark:text-emerald-400">
            {t('wechatBound')}
          </p>
          {accountID && (
            <p className="font-mono text-xs text-on-surface-variant">{accountID}</p>
          )}
          <Button variant="outline" size="sm" onClick={handleRebind} className="mt-1 gap-2">
            <RefreshCw className="h-3.5 w-3.5" />
            {t('wechatRebind')}
          </Button>
        </div>
      )
    }

    if (bindState === 'expired') {
      return (
        <div className="flex flex-col items-center gap-4 py-4">
          <div className="flex h-14 w-14 items-center justify-center rounded-full bg-amber-500/10">
            <X className="h-7 w-7 text-amber-600 dark:text-amber-400" />
          </div>
          <p className="text-sm text-amber-600 dark:text-amber-400">{t('wechatExpired')}</p>
          <Button variant="outline" size="sm" onClick={() => void handleBind()} className="gap-2">
            <RefreshCw className="h-3.5 w-3.5" />
            {t('wechatRetry')}
          </Button>
        </div>
      )
    }

    if (bindState === 'error') {
      return (
        <div className="flex flex-col items-center gap-4 py-4">
          <div className="flex h-14 w-14 items-center justify-center rounded-full bg-error-container/50">
            <X className="h-7 w-7 text-error" />
          </div>
          <p className="text-sm text-error">{bindError || t('wechatBindError')}</p>
          <Button variant="outline" size="sm" onClick={handleRebind} className="gap-2">
            <RefreshCw className="h-3.5 w-3.5" />
            {t('wechatRetry')}
          </Button>
        </div>
      )
    }

    return null
  }

  return (
    <div className="space-y-6">
      <Card className="rounded-xl border-outline-variant/20 bg-surface-container-lowest shadow-sm">
        <CardHeader className="pb-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <MessageSquare className="h-5 w-5 text-primary" />
              <div>
                <CardTitle className="text-lg font-bold tracking-tight">{t('wechatBotTitle')}</CardTitle>
                <CardDescription>{t('wechatBotDescription')}</CardDescription>
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

          {/* QR Code Binding Section */}
          {renderBindSection()}

          <Separator />

          {/* Connection status indicator */}
          <div className="flex items-center justify-between gap-3 flex-wrap">
            <div className="flex items-center gap-3">
              <div className={cn(
                'h-3 w-3 rounded-full',
                loading ? 'bg-on-surface-variant/30' :
                isConnected ? 'bg-primary' : 'bg-error'
              )} />
              <div className="space-y-0.5">
                <p className="text-sm font-medium text-on-surface">
                  {loading ? t('wechatStatusChecking') :
                   isConnected ? t('wechatConnected') : t('wechatDisconnected')}
                </p>
                {!loading && isConnected && nickname && (
                  <p className="text-xs text-on-surface-variant">
                    {t('wechatSelfHint', { nickname })}
                  </p>
                )}
              </div>
            </div>
            <Badge variant={isConnected ? 'outline' : 'secondary'}>
              {isConnected ? t('wechatBadgeConnected') : t('wechatBadgeDisconnected')}
            </Badge>
          </div>

          {/* Error message */}
          {error && (
            <div className="rounded-lg bg-error-container px-3 py-2 text-sm text-error">
              {error}
            </div>
          )}

          {/* Action buttons */}
          <div className="flex items-center gap-3 flex-wrap">
            {!isConnected ? (
              <Button
                variant="default"
                size="sm"
                onClick={() => void handleConnect()}
                disabled={connecting || tokenExpired}
              >
                {connecting ? (
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                ) : (
                  <RefreshCw className="mr-2 h-4 w-4" />
                )}
                {connecting ? t('wechatConnecting') : t('wechatConnect')}
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
                {t('wechatDisconnect')}
              </Button>
            )}
            <Button
              variant="outline"
              size="sm"
              onClick={() => refreshStatus()}
              disabled={loading}
            >
              <RefreshCw className={cn('mr-2 h-4 w-4', loading ? 'animate-spin' : '')} />
              {t('common:refresh')}
            </Button>
          </div>

          {/* Hint text when not enabled */}
          {!isEnabled && !isConnected && (
            <p className="text-xs text-on-surface-variant">
              {t('wechatNotEnabledHint')}
            </p>
          )}
          {/* Hint text when token expired */}
          {tokenExpired && (
            <p className="text-xs text-red-500">
              {t('wechatTokenExpiredHint')}
            </p>
          )}
        </CardContent>
        )}
      </Card>
    </div>
  )
}
