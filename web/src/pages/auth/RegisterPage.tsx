import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Link, useNavigate } from 'react-router-dom'
import { Warehouse } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from '@/components/ui/card'
import { useAuthStore } from '@/store/authStore'

export default function RegisterPage() {
  const { t } = useTranslation('auth')
  const navigate = useNavigate()
  const register = useAuthStore((s) => s.register)
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [displayName, setDisplayName] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      await register(username, password, displayName)
      navigate('/')
    } catch (err) {
      setError(err instanceof Error ? err.message : t('registerFailed'))
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-surface p-4">
      <Card className="w-full max-w-sm">
        <CardHeader className="text-center">
          <div className="flex justify-center mb-4">
            <div className="w-12 h-12 rounded-xl bg-gradient-primary flex items-center justify-center">
              <Warehouse className="w-7 h-7 text-on-primary" />
            </div>
          </div>
          <CardTitle>{t('registerTitle')}</CardTitle>
          <CardDescription>{t('registerSubtitle')}</CardDescription>
        </CardHeader>
        <form onSubmit={handleSubmit}>
          <CardContent className="space-y-4">
            {error && (
              <div className="text-sm text-error bg-error-container/20 rounded-lg p-3">{error}</div>
            )}
            <div className="space-y-2">
              <label className="text-sm font-medium text-on-surface" htmlFor="username">
                {t('username')}
              </label>
              <Input
                id="username"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                placeholder={t('usernamePlaceholder')}
                autoComplete="username"
                required
              />
            </div>
            <div className="space-y-2">
              <label className="text-sm font-medium text-on-surface" htmlFor="displayName">
                {t('displayName')}
              </label>
              <Input
                id="displayName"
                value={displayName}
                onChange={(e) => setDisplayName(e.target.value)}
                placeholder={t('displayNamePlaceholder')}
              />
            </div>
            <div className="space-y-2">
              <label className="text-sm font-medium text-on-surface" htmlFor="password">
                {t('password')}
              </label>
              <Input
                id="password"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder={t('passwordPlaceholder')}
                autoComplete="new-password"
                required
              />
            </div>
          </CardContent>
          <CardFooter className="flex-col gap-3">
            <Button type="submit" className="w-full" disabled={loading}>
              {loading ? t('registering') : t('register')}
            </Button>
            <p className="text-sm text-on-surface-variant">
              {t('hasAccount')}{' '}
              <Link to="/login" className="text-primary hover:underline">{t('loginLink')}</Link>
            </p>
          </CardFooter>
        </form>
      </Card>
    </div>
  )
}
