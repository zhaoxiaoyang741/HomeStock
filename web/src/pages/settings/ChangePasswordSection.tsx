import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Lock, Loader2, Save } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Separator } from '@/components/ui/separator'
import { authApi } from '@/api/auth'

export function ChangePasswordSection() {
  const { t } = useTranslation('settings')
  const [oldPassword, setOldPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')

  const reset = () => {
    setOldPassword('')
    setNewPassword('')
    setConfirmPassword('')
  }

  async function handleSave() {
    setError('')
    setSuccess('')

    if (newPassword !== confirmPassword) {
      setError(t('passwordMismatch'))
      return
    }

    setSaving(true)
    try {
      await authApi.changePassword({
        old_password: oldPassword,
        new_password: newPassword,
      })
      setSuccess(t('passwordChangeSuccess'))
      reset()
    } catch (err) {
      const msg = err instanceof Error ? err.message : String(err)
      setError(msg || t('passwordChangeFailed'))
    } finally {
      setSaving(false)
    }
  }

  return (
    <Card className="rounded-xl border-outline-variant/20 bg-surface-container-lowest shadow-sm">
      <CardHeader className="pb-4">
        <div className="flex items-center gap-3">
          <Lock className="w-5 h-5 text-primary" />
          <div>
            <CardTitle className="text-lg font-bold tracking-tight">{t('passwordSectionTitle')}</CardTitle>
            <CardDescription>{t('passwordSectionDescription')}</CardDescription>
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

        {success && (
          <div className="rounded-lg bg-primary/10 px-3 py-2 text-sm text-primary">
            {success}
          </div>
        )}

        <div className="space-y-1.5">
          <Label htmlFor="old-password">{t('passwordOldLabel')}</Label>
          <Input
            id="old-password"
            type="password"
            value={oldPassword}
            onChange={(e) => setOldPassword(e.target.value)}
            placeholder={t('passwordOldPlaceholder')}
          />
        </div>

        <div className="space-y-1.5">
          <Label htmlFor="new-password">{t('passwordNewLabel')}</Label>
          <Input
            id="new-password"
            type="password"
            value={newPassword}
            onChange={(e) => setNewPassword(e.target.value)}
            placeholder={t('passwordNewPlaceholder')}
          />
        </div>

        <div className="space-y-1.5">
          <Label htmlFor="confirm-password">{t('passwordConfirmLabel')}</Label>
          <Input
            id="confirm-password"
            type="password"
            value={confirmPassword}
            onChange={(e) => setConfirmPassword(e.target.value)}
            placeholder={t('passwordConfirmPlaceholder')}
          />
        </div>

        <Button
          size="sm"
          onClick={() => void handleSave()}
          disabled={saving || !oldPassword || !newPassword || !confirmPassword}
        >
          {saving ? (
            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
          ) : (
            <Save className="mr-2 h-4 w-4" />
          )}
          {saving ? t('passwordChanging') : t('common:save')}
        </Button>
      </CardContent>
    </Card>
  )
}
