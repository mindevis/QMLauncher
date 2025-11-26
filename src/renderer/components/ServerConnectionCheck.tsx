import { useState, useEffect } from 'react'
import { Loader2 } from 'lucide-react'
import { API_BASE_URL } from '../config/api'
import { useI18n } from '../contexts/I18nContext'

interface ServerConnectionCheckProps {
  onServerAvailable: () => void
  onServerUnavailable: (error: string) => void
}

export function ServerConnectionCheck({ onServerAvailable, onServerUnavailable }: ServerConnectionCheckProps) {
  const { t } = useI18n()
  const [isChecking, setIsChecking] = useState(true)

  useEffect(() => {
    checkServerConnection()
  }, [])

  const checkServerConnection = async () => {
    try {
      setIsChecking(true)

      // Try to connect to API health endpoint or servers endpoint
      const response = await fetch(`${API_BASE_URL}/servers`, {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
        },
        // Add timeout
        signal: AbortSignal.timeout(5000)
      })

      if (response.ok) {
        onServerAvailable()
      } else {
        throw new Error(`Server returned status ${response.status}`)
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : t('common.unknownError')
      const fullErrorMessage = t('error.connectionFailed', { error: errorMessage })
      onServerUnavailable(fullErrorMessage)
    } finally {
      setIsChecking(false)
    }
  }

  if (isChecking) {
    return (
      <div className="flex flex-col items-center justify-center h-full p-8">
        <Loader2 className="w-12 h-12 animate-spin mb-4 text-primary" />
        <p className="text-lg font-medium text-foreground">
          {t('connection.checking')}
        </p>
        <p className="text-sm mt-2 text-muted-foreground">
          {API_BASE_URL}
        </p>
      </div>
    )
  }

  // Не показываем ошибку здесь, она будет показана в модальном окне
  // Просто возвращаем null, чтобы форма авторизации была видна за модальным окном
  return null
}

