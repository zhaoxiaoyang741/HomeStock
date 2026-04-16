import { useTranslation } from 'react-i18next'
import { Settings2 } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Checkbox } from '@/components/ui/checkbox'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { NotificationHistorySection } from './NotificationHistorySection'

type FormState = {
  remindDays: string
  checkTime: string
  notifyEnabled: boolean
}

type Props = {
  form: FormState
  onChange: (patch: Partial<FormState>) => void
}

export function ReminderSection({ form, onChange }: Props) {
  const { t } = useTranslation('settings')

  return (
    <div className="space-y-6">
      <Card className="rounded-xl border-outline-variant/20 bg-surface-container-lowest shadow-sm">
        <CardHeader className="pb-4">
          <div className="flex items-center gap-3">
            <div className="flex h-11 w-11 items-center justify-center rounded-full bg-primary/10 text-primary">
              <Settings2 className="h-5 w-5" />
            </div>
            <div>
              <CardTitle className="text-lg font-bold tracking-tight">{t('reminderTitle')}</CardTitle>
              <CardDescription>{t('reminderDescription')}</CardDescription>
            </div>
          </div>
        </CardHeader>
        <CardContent className="space-y-5">
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label htmlFor="settings-remind-days">{t('fieldRemindDays')}</Label>
              <Input
                id="settings-remind-days"
                type="number"
                min={0}
                max={30}
                value={form.remindDays}
                onChange={(e) => onChange({ remindDays: e.target.value })}
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="settings-check-time">{t('fieldCheckTime')}</Label>
              <Input
                id="settings-check-time"
                type="time"
                value={form.checkTime}
                onChange={(e) => onChange({ checkTime: e.target.value })}
              />
            </div>
          </div>
          <div className="rounded-lg bg-surface-container px-4 py-3 text-sm text-on-surface-variant">
            {t('reminderHint')}
          </div>
        </CardContent>
      </Card>

      <Card className="rounded-xl border-outline-variant/20 bg-surface-container-lowest shadow-sm">
        <CardHeader className="pb-4">
          <CardTitle className="text-lg font-bold tracking-tight">{t('notifyTitle')}</CardTitle>
          <CardDescription>{t('notifyDescription')}</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex items-center justify-between rounded-lg border border-outline-variant/20 bg-surface-container-low px-4 py-3">
            <div>
              <p className="text-sm font-medium text-on-surface">{t('fieldNotifyEnabled')}</p>
              <p className="text-xs text-on-surface-variant">{t('notifyEnabledHint')}</p>
            </div>
            <Checkbox
              checked={form.notifyEnabled}
              onCheckedChange={(checked) => onChange({ notifyEnabled: checked === true })}
            />
          </div>
        </CardContent>
      </Card>

      <NotificationHistorySection />
    </div>
  )
}
