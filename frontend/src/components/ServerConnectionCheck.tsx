import { useState, useEffect } from 'react'
import { Loader2 } from 'lucide-react'
import { API_BASE_URL } from '../config/api'
import { useI18n } from '../contexts/I18nContext'
import { api } from '../utils/api-client'

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
      console.log(`[ServerConnectionCheck] Attempting to connect to: ${API_BASE_URL}/servers`)

      const controller = new AbortController()
      const timeoutId = setTimeout(() => controller.abort(), 10000)

      try {
        console.log('[ServerConnectionCheck] Checking server connection')
        
        const response = await api.get('/servers')

        clearTimeout(timeoutId)
        console.log(`[ServerConnectionCheck] Response status: ${response.status}`)

        if (response.ok) {
          console.log('[ServerConnectionCheck] Connection successful')
          onServerAvailable()
        } else {
          const errorText = typeof response.data === 'string' ? response.data : JSON.stringify(response.data)
          console.error(`[ServerConnectionCheck] Server returned error: ${response.status}`, errorText)
          throw new Error(`Server returned status ${response.status}`)
        }
      } catch (fetchError) {
        clearTimeout(timeoutId)
        if (fetchError instanceof Error && fetchError.name === 'AbortError') {
          throw new Error('Connection timeout after 10 seconds')
        }
        throw fetchError
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : t('common.unknownError')
      console.error(`[ServerConnectionCheck] Connection failed:`, err)
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

  return null
}

