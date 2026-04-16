import { useTranslation } from 'react-i18next'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Separator } from '@/components/ui/separator'
import type { SystemSettings } from '@/types/systemSettings'

type FormState = {
  feishuWebhookInput: string
  webhookMode: 'keep' | 'replace' | 'clear'
}

type Props = {
  form: FormState
  initialSettings: SystemSettings
  onChange: (patch: Partial<FormState>) => void
}

export function ChannelSection({ form, initialSettings, onChange }: Props) {
  const { t } = useTranslation('settings')

  function handleWebhookInputChange(value: string) {
    onChange({
      feishuWebhookInput: value,
      webhookMode: value.trim() ? 'replace' : 'keep',
    })
  }

  function handleClear() {
    onChange({ feishuWebhookInput: '', webhookMode: 'clear' })
  }

  return (
    <div className="space-y-6">
      <Card className="rounded-xl border-outline-variant/20 bg-surface-container-lowest shadow-sm">
        <CardHeader className="pb-4">
          <CardTitle className="text-lg font-bold tracking-tight">{t('notifyTitle')}</CardTitle>
          <CardDescription>{t('notifyDescription')}</CardDescription>
        </CardHeader>
        <CardContent className="space-y-5">
          <Separator />
          <div className="space-y-3">
            <div className="flex items-center justify-between gap-3 flex-wrap">
              <div className="space-y-1">
                <Label htmlFor="settings-webhook">{t('fieldWebhook')}</Label>
                <p className="text-xs text-on-surface-variant">{t('webhookHint')}</p>
              </div>
              <Badge variant={initialSettings.notify.feishu_webhook_configured ? 'outline' : 'secondary'}>
                {initialSettings.notify.feishu_webhook_configured
                  ? t('webhookConfigured')
                  : t('webhookNotConfigured')}
              </Badge>
            </div>

            {initialSettings.notify.feishu_webhook_configured && (
              <div className="rounded-lg bg-surface-container px-3 py-2 text-sm text-on-surface-variant">
                {t('webhookMaskedLabel', { value: initialSettings.notify.feishu_webhook_masked })}
              </div>
            )}

            <Input
              id="settings-webhook"
              type="url"
              value={form.feishuWebhookInput}
              placeholder={t('webhookPlaceholder')}
              onChange={(e) => handleWebhookInputChange(e.target.value)}
            />

            <div className="flex items-center gap-2 flex-wrap">
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={handleClear}
                disabled={!initialSettings.notify.feishu_webhook_configured && form.feishuWebhookInput.trim() === ''}
              >
                {t('clearWebhook')}
              </Button>
              {form.webhookMode === 'clear' && (
                <span className="text-xs text-error">{t('webhookClearPending')}</span>
              )}
              {form.webhookMode === 'replace' && (
                <span className="text-xs text-primary">{t('webhookReplacePending')}</span>
              )}
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
