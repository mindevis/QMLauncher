import { useState } from 'react'
import { User, Lock, ExternalLink } from 'lucide-react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from './ui/card'
import { api } from '../utils/api-client'
import { Input } from './ui/input'
import { Label } from './ui/label'
import { Button } from './ui/button'
import { Checkbox } from './ui/checkbox'
import { QMLogo } from './QMLogo'
import { useI18n } from '../contexts/I18nContext'

interface LoginFormProps {
  onLoginSuccess: (token: string) => void
  qmWebUrl?: string
}

export function LoginForm({ onLoginSuccess, qmWebUrl = 'https://qmweb.example.com' }: LoginFormProps) {
  const { t } = useI18n()
  const [login, setLogin] = useState('')
  const [password, setPassword] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [showRegistrationLink, setShowRegistrationLink] = useState(false)
  const [rememberSession, setRememberSession] = useState(true) // По умолчанию включено

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setIsLoading(true)
    setError(null)
    setShowRegistrationLink(false)

    try {
      const response = await api.post<{ access_token: string } | { detail: string | any[]; message?: string }>('/auth/login', {
        email: login,
        password,
      })

      if (response.ok && 'access_token' in response.data) {
        // Save token to localStorage only if "remember session" is checked
        if (rememberSession) {
          localStorage.setItem('qmlauncher_auth_token', response.data.access_token)
        } else {
          // Если не сохранять сессию, удаляем токен если он был сохранен ранее
          localStorage.removeItem('qmlauncher_auth_token')
        }
        onLoginSuccess(response.data.access_token)
      } else {
        const errorData = response.data as { detail?: string | any[]; message?: string } || { detail: t('login.authError') }
        
        let errorMessage: string
        if (Array.isArray(errorData.detail)) {
          errorMessage = errorData.detail
            .map((err: any) => {
              if (typeof err === 'string') return err
              if (err.msg) return err.msg
              return JSON.stringify(err)
            })
            .join(', ')
        } else if (typeof errorData.detail === 'string') {
          errorMessage = errorData.detail
        } else if (errorData.message) {
          errorMessage = errorData.message
        } else {
          errorMessage = t('login.authError')
        }
        
        if (errorMessage.includes('Incorrect') || errorMessage.includes('not found')) {
          setShowRegistrationLink(true)
        }
        
        setError(errorMessage)
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : t('login.connectionError')
      console.error('[LoginForm] Login error:', err)
      setError(errorMessage)
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <div className="flex items-center justify-center h-full p-12">
      <Card className="w-full max-w-md no-drag shadow-lg p-6">
        <CardHeader className="space-y-1 px-0">
          <div className="flex justify-center mb-2">
            <QMLogo className="h-12 w-12 dark:invert" />
          </div>
          <CardTitle className="text-2xl font-bold text-center">{t('login.title')}</CardTitle>
          <CardDescription className="text-center">
            {t('login.description')}
          </CardDescription>
        </CardHeader>
        <CardContent className="px-0">
          <form onSubmit={handleSubmit} className="space-y-4">
            {error && (
              <div className="rounded-md bg-destructive/15 p-3 text-sm text-destructive">
                {error}
                {showRegistrationLink && (
                  <a
                    href={qmWebUrl}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="flex items-center gap-2 mt-2 text-sm underline text-primary"
                  >
                    <ExternalLink className="w-4 h-4" />
                    {t('login.registerLink')}
                  </a>
                )}
              </div>
            )}

            <div className="space-y-2">
              <Label htmlFor="login">{t('login.loginLabel')}</Label>
              <div className="relative">
                <User className="absolute left-3 top-1/2 transform -translate-y-1/2 w-5 h-5 text-muted-foreground pointer-events-none" />
                <Input
                  id="login"
                  type="text"
                  placeholder={t('login.loginPlaceholder')}
                  value={login}
                  onChange={(e) => setLogin(e.target.value)}
                  required
                  disabled={isLoading}
                  className="pl-10"
                />
              </div>
            </div>
            <div className="space-y-2">
              <Label htmlFor="password">{t('login.passwordLabel')}</Label>
              <div className="relative">
                <Lock className="absolute left-3 top-1/2 transform -translate-y-1/2 w-5 h-5 text-muted-foreground pointer-events-none" />
                <Input
                  id="password"
                  type="password"
                  placeholder={t('login.passwordPlaceholder')}
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  required
                  disabled={isLoading}
                  className="pl-10"
                />
              </div>
            </div>

            <div className="flex items-center space-x-2">
              <Checkbox
                id="rememberSession"
                checked={rememberSession}
                onCheckedChange={(checked: boolean) => setRememberSession(checked === true)}
              />
              <Label
                htmlFor="rememberSession"
                className="text-sm font-normal cursor-pointer"
              >
                {t('login.rememberSession') || 'Сохранить сессию'}
              </Label>
            </div>

            <Button type="submit" className="w-full" disabled={isLoading}>
              {isLoading ? t('login.submitting') : t('login.submitButton')}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  )
}
