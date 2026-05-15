import { useEffect, useState, useCallback, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import { MessageSquare, PowerOff, RefreshCw, Loader2, QrCode, Smartphone, CheckCircle2, AlertCircle } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Separator } from '@/components/ui/separator'
import { cn } from '@/lib/utils'
import { getWechatQrCode, getWechatStatus, disconnectWechat, reconnectWechat } from '@/api/wechat'
import type { WechatStatus, WechatQrCode } from '@/types/wechat'

export function WechatBotSection() {
  const { t } = useTranslation('settings')
  const [status, setStatus] = useState<WechatStatus | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [connecting, setConnecting] = useState(false)
  const [disconnecting, setDisconnecting] = useState(false)

  // QR code state
  const [qrCode, setQrCode] = useState<WechatQrCode | null>(null)
  const [qrPolling, setQrPolling] = useState(false)
  const qrPollRef = useRef<ReturnType<typeof setInterval> | null>(null)

  const pollingRef = useRef<ReturnType<typeof setInterval> | null>(null)

  const fetchStatus = useCallback(async () => {
    try {
      const s = await getWechatStatus()
      setStatus(s)
      setError('')
    } catch (err) {
      if (!loading) {
        setError(err instanceof Error ? err.message : t('wechatStatusFailed'))
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
      if (pollingRef.current) clearInterval(pollingRef.current)
    }
  }, [fetchStatus])

  // QR code polling
  const startQrPolling = useCallback(() => {
    setQrPolling(true)
    const poll = async () => {
      try {
        const qr = await getWechatQrCode()
        setQrCode(qr)

        // If login successful or expired, stop polling
        if (qr.status === 'success' || qr.status === 'expired') {
          if (qrPollRef.current) clearInterval(qrPollRef.current)
          setQrPolling(false)
          // Refresh overall status
          void fetchStatus()
        }
      } catch {
        // ignore polling errors
      }
    }
    void poll()
    qrPollRef.current = setInterval(poll, 2000)
  }, [fetchStatus])

  const stopQrPolling = useCallback(() => {
    if (qrPollRef.current) {
      clearInterval(qrPollRef.current)
      qrPollRef.current = null
    }
    setQrPolling(false)
    setQrCode(null)
  }, [])

  // Cleanup QR polling on unmount
  useEffect(() => {
    return () => {
      if (qrPollRef.current) clearInterval(qrPollRef.current)
    }
  }, [])

  async function handleConnect() {
    setConnecting(true)
    setError('')
    try {
      await reconnectWechat()
      // Start polling QR code
      startQrPolling()
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
      stopQrPolling()
      setStatus((prev) => prev ? { ...prev, connected: false, logged_in: false } : null)
    } catch (err) {
      setError(err instanceof Error ? err.message : t('wechatDisconnectFailed'))
    } finally {
      setDisconnecting(false)
    }
  }

  const isConnected = status?.connected ?? false
  const isLoggedIn = status?.logged_in ?? false
  const isEnabled = status?.enabled ?? false
  const qrImageUrl = qrCode?.uuid ? `https://login.weixin.qq.com/qrcode/${qrCode.uuid}` : null

  return (
    <div className="space-y-6">
      <Card className="rounded-xl border-outline-variant/20 bg-surface-container-lowest shadow-sm">
        <CardHeader className="pb-4">
          <div className="flex items-center gap-3">
            <MessageSquare className="w-5 h-5 text-primary" />
            <div>
              <CardTitle className="text-lg font-bold tracking-tight">{t('wechatBotTitle')}</CardTitle>
              <CardDescription>{t('wechatBotDescription')}</CardDescription>
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
                  {loading ? t('wechatStatusChecking') :
                   isConnected ? t('wechatConnected') : t('wechatDisconnected')}
                </p>
                {!loading && isLoggedIn && (
                  <p className="text-xs text-on-surface-variant">{t('wechatLoggedIn')}</p>
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

          {/* QR code display */}
          {qrPolling && qrCode && qrCode.status !== 'success' && (
            <div className="flex flex-col items-center gap-3 py-4">
              {qrImageUrl && (
                <div className="relative">
                  <img
                    src={qrImageUrl}
                    alt="WeChat QR Code"
                    className="w-48 h-48 rounded-lg border border-outline-variant/20"
                  />
                  {/* Status overlay */}
                  {qrCode.status === 'scanned' && (
                    <div className="absolute inset-0 flex items-center justify-center rounded-lg bg-surface/80">
                      <div className="text-center">
                        <Smartphone className="w-8 h-8 mx-auto text-primary mb-2" />
                        <p className="text-sm font-medium text-on-surface">{t('wechatScanned')}</p>
                      </div>
                    </div>
                  )}
                  {qrCode.status === 'expired' && (
                    <div className="absolute inset-0 flex items-center justify-center rounded-lg bg-surface/80">
                      <div className="text-center">
                        <AlertCircle className="w-8 h-8 mx-auto text-error mb-2" />
                        <p className="text-sm font-medium text-on-surface">{t('wechatExpired')}</p>
                      </div>
                    </div>
                  )}
                </div>
              )}

              {/* Status text */}
              <div className="flex items-center gap-2 text-sm text-on-surface-variant">
                {qrCode.status === 'waiting' && (
                  <>
                    <QrCode className="w-4 h-4" />
                    <span>{t('wechatQrWaiting')}</span>
                  </>
                )}
                {qrCode.status === 'scanned' && (
                  <>
                    <Loader2 className="w-4 h-4 animate-spin" />
                    <span>{t('wechatQrScanned')}</span>
                  </>
                )}
              </div>

              {/* Retry button for expired */}
              {qrCode.status === 'expired' && (
                <Button
                  variant="default"
                  size="sm"
                  onClick={() => void handleConnect()}
                  disabled={connecting}
                >
                  {connecting ? (
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  ) : (
                    <RefreshCw className="mr-2 h-4 w-4" />
                  )}
                  {t('wechatRetry')}
                </Button>
              )}
            </div>
          )}

          {/* Login success */}
          {qrCode?.status === 'success' && !isConnected && (
            <div className="flex items-center gap-2 rounded-lg bg-primary/10 px-3 py-2 text-sm text-primary">
              <CheckCircle2 className="w-4 h-4" />
              <span>{t('wechatLoginSuccess')}</span>
            </div>
          )}

          {/* Action buttons */}
          <div className="flex items-center gap-3 flex-wrap">
            {!isConnected ? (
              <Button
                variant="default"
                size="sm"
                onClick={() => void handleConnect()}
                disabled={connecting || qrPolling}
              >
                {connecting || qrPolling ? (
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                ) : (
                  <Smartphone className="mr-2 h-4 w-4" />
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
              onClick={() => { setLoading(true); void fetchStatus() }}
              disabled={loading}
            >
              <RefreshCw className={cn('mr-2 h-4 w-4', loading ? 'animate-spin' : '')} />
              {t('common:refresh')}
            </Button>

            {/* Cancel QR polling */}
            {qrPolling && (
              <Button
                variant="ghost"
                size="sm"
                onClick={stopQrPolling}
              >
                {t('wechatCancel')}
              </Button>
            )}
          </div>

          {/* Hint text when not enabled */}
          {!isEnabled && !isConnected && (
            <p className="text-xs text-on-surface-variant">{t('wechatNotEnabledHint')}</p>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
