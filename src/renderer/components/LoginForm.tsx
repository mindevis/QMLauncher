import { useState } from 'react'
import { User, Lock, ExternalLink } from 'lucide-react'
import { API_BASE_URL } from '../config/api'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from './ui/card'
import { Input } from './ui/input'
import { Label } from './ui/label'
import { Button } from './ui/button'
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

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setIsLoading(true)
    setError(null)
    setShowRegistrationLink(false)

    try {
      // Авторизация через QMAdmin аккаунт
      const response = await fetch(`${API_BASE_URL}/auth/login`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ username: login, password }),
      })

      if (response.ok) {
        const data = await response.json()
        // Save token to Electron storage
        if (window.electronAPI?.saveAuthToken) {
          await window.electronAPI.saveAuthToken(data.access_token)
        }
        onLoginSuccess(data.access_token)
      } else {
        const errorData = await response.json().catch(() => ({ detail: t('login.authError') }))
        
        // Обрабатываем ошибки валидации FastAPI (422)
        let errorMessage: string
        if (Array.isArray(errorData.detail)) {
          // Если detail - массив ошибок валидации
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
        
        // Check if user doesn't exist
        if (errorMessage.includes('Incorrect') || errorMessage.includes('not found')) {
          setShowRegistrationLink(true)
        }
        
        setError(errorMessage)
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : t('login.connectionError')
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

            <Button type="submit" className="w-full" disabled={isLoading}>
              {isLoading ? t('login.submitting') : t('login.submitButton')}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  )
}

