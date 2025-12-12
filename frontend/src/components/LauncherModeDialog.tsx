import { useState } from 'react'
import { Server, Cloud, Loader2 } from 'lucide-react'
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from './ui/dialog'
import { Button } from './ui/button'
import { Input } from './ui/input'
import { Label } from './ui/label'
import { useI18n } from '../contexts/I18nContext'
import { toast } from 'sonner'

interface LauncherModeDialogProps {
  open: boolean
  onModeSelected: (mode: 'standalone' | 'server', serverUrl?: string) => void
}

export function LauncherModeDialog({ open, onModeSelected }: LauncherModeDialogProps) {
  const { t } = useI18n()
  const [selectedMode, setSelectedMode] = useState<'standalone' | 'server' | null>(null)
  const [serverUrl, setServerUrl] = useState('')
  const [isValidating, setIsValidating] = useState(false)
  const [urlError, setUrlError] = useState<string | null>(null)

  // Валидация URL сервера
  const validateServerUrl = (url: string): boolean => {
    if (!url.trim()) {
      setUrlError(t('modeDialog.serverUrlRequired') || 'Адрес сервера обязателен')
      return false
    }

    // Удаляем пробелы
    const cleanUrl = url.trim()

    // Проверяем различные форматы:
    // 1. Домен: example.com
    // 2. Домен с протоколом: http://example.com или https://example.com
    // 3. IP адрес: 192.168.1.1
    // 4. IP адрес с портом: 192.168.1.1:8000
    // 5. Домен с портом: example.com:8000
    // 6. Полный URL: http://example.com:8000/api/v1

    // Паттерн для валидации
    const domainPattern = /^([a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$/
    const ipPattern = /^(\d{1,3}\.){3}\d{1,3}(:\d+)?$/
    const urlPattern = /^https?:\/\/([a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}(:\d+)?(\/.*)?$/
    const ipUrlPattern = /^https?:\/\/(\d{1,3}\.){3}\d{1,3}(:\d+)?(\/.*)?$/

    // Проверяем различные форматы
    if (urlPattern.test(cleanUrl) || ipUrlPattern.test(cleanUrl)) {
      // Полный URL с протоколом
      setUrlError(null)
      return true
    } else if (domainPattern.test(cleanUrl) || ipPattern.test(cleanUrl)) {
      // Домен или IP без протокола
      setUrlError(null)
      return true
    } else {
      setUrlError(t('modeDialog.invalidServerUrl') || 'Неверный формат адреса сервера')
      return false
    }
  }

  const handleStandaloneMode = () => {
    toast.info(t('modeDialog.standaloneComingSoon') || 'Автономный режим будет доступен позже')
    // Пока не вызываем onModeSelected для standalone режима
  }

  const handleServerMode = async () => {
    if (!validateServerUrl(serverUrl)) {
      return
    }

    setIsValidating(true)
    setUrlError(null)

    try {
      // Нормализуем URL: добавляем протокол если его нет
      let normalizedUrl = serverUrl.trim()
      if (!normalizedUrl.startsWith('http://') && !normalizedUrl.startsWith('https://')) {
        // По умолчанию используем http, но можно попробовать https
        normalizedUrl = `http://${normalizedUrl}`
      }

      // Убираем /api/v1 если есть, добавим позже
      normalizedUrl = normalizedUrl.replace(/\/api\/v1\/?$/, '')
      
      // Проверяем доступность сервера
      const testUrl = `${normalizedUrl}/api/v1/health`
      const response = await fetch(testUrl, { 
        method: 'GET',
        mode: 'cors',
        signal: AbortSignal.timeout(5000) // 5 секунд таймаут
      })

      if (response.ok) {
        onModeSelected('server', normalizedUrl)
      } else {
        setUrlError(t('modeDialog.serverUnavailable') || 'Сервер недоступен')
      }
    } catch (error) {
      // Сервер может быть недоступен, но это не критично - пользователь может продолжить
      console.warn('Server validation error:', error)
      // Разрешаем продолжить даже если проверка не удалась
      onModeSelected('server', serverUrl.trim().startsWith('http') ? serverUrl.trim() : `http://${serverUrl.trim()}`)
    } finally {
      setIsValidating(false)
    }
  }

  const handleUrlChange = (value: string) => {
    setServerUrl(value)
    if (urlError) {
      setUrlError(null)
    }
  }

  return (
    <Dialog open={open} onOpenChange={() => {}}>
      <DialogContent className="sm:max-w-[500px] no-drag">
        <DialogHeader>
          <DialogTitle>{t('modeDialog.title') || 'Выбор режима работы'}</DialogTitle>
          <DialogDescription>
            {t('modeDialog.description') || 'Выберите режим работы лаунчера'}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-4">
          {/* Автономный режим */}
          <div
            className={`p-4 border-2 rounded-lg cursor-pointer transition-all ${
              selectedMode === 'standalone'
                ? 'border-primary bg-primary/10'
                : 'border-border hover:border-primary/50'
            }`}
            onClick={() => setSelectedMode('standalone')}
          >
            <div className="flex items-start gap-3">
              <Server className="w-6 h-6 mt-0.5 text-muted-foreground" />
              <div className="flex-1">
                <h3 className="font-semibold mb-1">
                  {t('modeDialog.standaloneTitle') || 'Автономный режим'}
                </h3>
                <p className="text-sm text-muted-foreground">
                  {t('modeDialog.standaloneDescription') || 'Работа без подключения к QMServer'}
                </p>
              </div>
            </div>
          </div>

          {/* Режим с подключением к QMServer */}
          <div
            className={`p-4 border-2 rounded-lg transition-all ${
              selectedMode === 'server'
                ? 'border-primary bg-primary/10'
                : 'border-border hover:border-primary/50'
            }`}
            onClick={() => setSelectedMode('server')}
          >
            <div className="flex items-start gap-3">
              <Cloud className="w-6 h-6 mt-0.5 text-muted-foreground" />
              <div className="flex-1">
                <h3 className="font-semibold mb-1">
                  {t('modeDialog.serverTitle') || 'Режим с подключением к QMServer'}
                </h3>
                <p className="text-sm text-muted-foreground mb-3">
                  {t('modeDialog.serverDescription') || 'Подключение к серверу QMServer для управления серверами'}
                </p>

                {selectedMode === 'server' && (
                  <div className="space-y-2 mt-3">
                    <Label htmlFor="serverUrl">
                      {t('modeDialog.serverUrlLabel') || 'Адрес QMServer'}
                    </Label>
                    <Input
                      id="serverUrl"
                      type="text"
                      placeholder={t('modeDialog.serverUrlPlaceholder') || 'example.com:8000 или http://example.com:8000'}
                      value={serverUrl}
                      onChange={(e) => handleUrlChange(e.target.value)}
                      onBlur={() => {
                        if (serverUrl) {
                          validateServerUrl(serverUrl)
                        }
                      }}
                      className={urlError ? 'border-destructive' : ''}
                    />
                    {urlError && (
                      <p className="text-sm text-destructive">{urlError}</p>
                    )}
                    <p className="text-xs text-muted-foreground">
                      {t('modeDialog.serverUrlHint') || 'Можно указать домен, IP адрес или полный URL'}
                    </p>
                  </div>
                )}
              </div>
            </div>
          </div>
        </div>

        <div className="flex justify-end gap-2">
          {selectedMode === 'standalone' && (
            <Button onClick={handleStandaloneMode} variant="default">
              {t('modeDialog.continue') || 'Продолжить'}
            </Button>
          )}
          {selectedMode === 'server' && (
            <Button 
              onClick={handleServerMode} 
              variant="default"
              disabled={isValidating || !serverUrl.trim()}
            >
              {isValidating ? (
                <>
                  <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                  {t('modeDialog.validating') || 'Проверка...'}
                </>
              ) : (
                t('modeDialog.continue') || 'Продолжить'
              )}
            </Button>
          )}
        </div>
      </DialogContent>
    </Dialog>
  )
}

