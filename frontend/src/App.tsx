import { useState, useEffect, lazy, Suspense } from 'react'
import { TitleBar } from './components/TitleBar'
import { ServerConnectionCheck } from './components/ServerConnectionCheck'
import { LoginForm } from './components/LoginForm'
import { ErrorModal } from './components/ErrorModal'
import { NavUser } from './components/NavUser'
import { QMLogo } from './components/QMLogo'
import { Server, Newspaper, Settings } from 'lucide-react'
import { motion } from 'framer-motion'
import { Button } from './components/ui/button'
import { Loader2 } from 'lucide-react'
import { wailsAPI } from './bridge'
import { api } from './utils/api-client'
import { I18nProvider, useI18n } from './contexts/I18nContext'
import { ThemeProvider } from './contexts/ThemeContext'
import { Toaster } from './components/ui/sonner'
import { ErrorBoundary } from './components/ErrorBoundary'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { LauncherModeDialog } from './components/LauncherModeDialog'
import './App.css'

// Создаем QueryClient с настройками кэширования
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30000, // 30 секунд - данные считаются свежими
      gcTime: 300000, // 5 минут - время хранения в кэше (было cacheTime)
      retry: 2, // Повторять запросы 2 раза при ошибке
      refetchOnWindowFocus: false, // Не обновлять при фокусе окна
    },
  },
})

// Lazy load tabs for code splitting
const ServersTab = lazy(() => import('./components/ServersTab').then(m => ({ default: m.ServersTab })))
const NewsTab = lazy(() => import('./components/NewsTab').then(m => ({ default: m.NewsTab })))
const SettingsTab = lazy(() => import('./components/SettingsTab').then(m => ({ default: m.SettingsTab })))

type Tab = 'servers' | 'news' | 'settings'
type AppState = 'checking' | 'login' | 'authenticated' | 'offline' | 'error'

function AppContent() {
  const { t } = useI18n()
  // Читаем токен один раз при инициализации (до первого рендера)
  const initialStoredToken = typeof window !== 'undefined' ? localStorage.getItem('qmlauncher_auth_token') : null

  const [activeTab, setActiveTab] = useState<Tab>('news')
  const [appVersion, setAppVersion] = useState<string>('')
  // Если есть сохранённый токен — стартуем сразу в authenticated (UI не мигает логином/проверкой)
  const [appState, setAppState] = useState<AppState>(initialStoredToken ? 'authenticated' : 'checking')
  const [authToken, setAuthToken] = useState<string | null>(initialStoredToken)
  const [connectionError, setConnectionError] = useState<string | null>(null)
  const [showErrorModal, setShowErrorModal] = useState(false)
  const [showModeDialog, setShowModeDialog] = useState(false)
  const [isCheckingFirstLaunch, setIsCheckingFirstLaunch] = useState(true)

  useEffect(() => {
    console.log('[App] Component mounted, initializing...')
    
    // Check if this is first launch
    const checkFirstLaunch = async () => {
      try {
        const dirExists = await wailsAPI.checkQMLauncherDirExists()
        console.log('[App] .qmlauncher directory exists:', dirExists)
        if (!dirExists) {
          console.log('[App] First launch detected, showing mode selection dialog')
          setShowModeDialog(true)
        }
      } catch (error) {
        console.error('[App] Error checking .qmlauncher directory:', error)
        // On error, assume it's not first launch
      } finally {
        setIsCheckingFirstLaunch(false)
      }
    }
    
    checkFirstLaunch()
    
    // Get version
    wailsAPI.getAppVersion()
      .then((version) => {
        console.log('[App] Version:', version)
        setAppVersion(version)
      })
      .catch((err) => {
        console.error('[App] Error getting version:', err)
      })

    // Listen for navigation events
    const handleNavigation = (e: Event) => {
      const customEvent = e as CustomEvent<string>
      const page = customEvent.detail
      if (page === 'settings' || page === 'servers' || page === 'news') {
        setActiveTab(page as Tab)
      }
    }
    
    window.addEventListener('navigate', handleNavigation)
    
    // Если токен уже загружен (initialStoredToken), запускаем фоновую валидацию
    if (initialStoredToken) {
      console.log('[App] Using initial stored token, background validation will run')
      setTimeout(() => validateTokenInBackground(initialStoredToken), 100)
    } else {
      console.log('[App] No auth token found, will check server connection')
    }

    return () => {
      window.removeEventListener('navigate', handleNavigation)
    }
  }, [])

  // Функция для валидации токена в фоне
  const validateTokenInBackground = async (token: string) => {
    try {
      const response = await api.get('/auth/me', { authToken: token })
      if (response.ok) {
        console.log('[App] Auth token validated, user authenticated')
        setAppState('authenticated')
        // Скрываем ошибку подключения если она была показана
        setShowErrorModal(false)
        setConnectionError(null)
      } else if (response.status === 401) {
        console.log('[App] Auth token is unauthorized (401), clearing...')
        localStorage.removeItem('qmlauncher_auth_token')
        setAuthToken(null)
        setAppState('login')
      } else {
        // Серверная ошибка (не 401) - оставляем пользователя авторизованным
        console.log('[App] Server error during token validation (non-401), keeping token and switching to offline')
        setAppState('offline')
        // Показываем предупреждение о проблеме с сервером
        const errorMsg = `Ошибка сервера (${response.status}): ${response.statusText || 'Неизвестная ошибка'}`
        setConnectionError(errorMsg)
        setShowErrorModal(true)
      }
    } catch (error) {
      console.error('[App] Error validating token after server check:', error)
      const errorMessage = error instanceof Error ? error.message : String(error)
      if (errorMessage.includes('401') || errorMessage.includes('Unauthorized')) {
        console.log('[App] Token is unauthorized, clearing...')
        localStorage.removeItem('qmlauncher_auth_token')
        setAuthToken(null)
        setAppState('login')
      } else {
        // Сетевая ошибка - оставляем пользователя авторизованным, но режим offline
        console.log('[App] Network/connection error during validation, keeping token and switching to offline')
        setAppState('offline')
        // Показываем предупреждение о проблеме с подключением
        setConnectionError(`Ошибка подключения: ${errorMessage}`)
        setShowErrorModal(true)
      }
    }
  }

  const handleServerAvailable = async () => {
    console.log('[App] Server is available')
    // If we have a token, validate it now that server is available
    if (authToken) {
      await validateTokenInBackground(authToken)
    } else {
      setAppState('login')
    }
  }

  const handleServerUnavailable = (error: string) => {
    console.error('[App] Server unavailable:', error)
    setConnectionError(error)
    setShowErrorModal(true)
    // Если есть сохраненный токен, оставляем пользователя авторизованным
    // Показываем ошибку подключения, но не требуем повторного входа
    if (authToken) {
      console.log('[App] Server unavailable but token exists, switching to offline')
      setAppState('offline')
    } else {
      setAppState('login')
    }
  }

  const handleRetryConnection = () => {
    setShowErrorModal(false)
    setConnectionError(null)
    // Если есть токен, переходим в checking для повторной проверки сервера и валидации токена
    // Если токена нет, переходим в login
    if (authToken) {
      setAppState('checking')
    } else {
      setAppState('login')
    }
  }

  const handleCloseErrorModal = () => {
    setShowErrorModal(false)
    // Не очищаем connectionError, чтобы можно было снова показать при необходимости
  }

  const handleLoginSuccess = (token: string) => {
    setAuthToken(token)
    setAppState('authenticated')
  }

  const handleLogout = () => {
    localStorage.removeItem('qmlauncher_auth_token')
    setAuthToken(null)
    setAppState('login')
  }

  const handleModeSelected = async (mode: 'standalone' | 'server', serverUrl?: string) => {
    console.log('[App] Mode selected:', mode, serverUrl)
    setShowModeDialog(false)
    
    if (mode === 'server' && serverUrl) {
      // Сохраняем URL сервера в настройках
      try {
        const settings = await wailsAPI.getSettings()
        if (settings) {
          // Нормализуем URL: добавляем /api/v1 если его нет
          let normalizedUrl = serverUrl
          if (!normalizedUrl.endsWith('/api/v1')) {
            normalizedUrl = normalizedUrl.replace(/\/api\/v1\/?$/, '')
            normalizedUrl = `${normalizedUrl}/api/v1`
          }
          
          settings.apiBaseUrl = normalizedUrl
          await wailsAPI.saveSettings(settings)
          console.log('[App] Server URL saved:', normalizedUrl)
        }
      } catch (error) {
        console.error('[App] Error saving server URL:', error)
      }
      
      // После выбора режима с сервером, проверяем подключение
      setAppState('checking')
    } else {
      // Для standalone режима пока просто закрываем диалог
      // В будущем здесь будет логика для автономного режима
    }
  }



  return (
    <div className="h-screen flex flex-col overflow-hidden bg-transparent">
      <div className="flex-1 flex flex-col overflow-hidden">
        {/* Title Bar */}
        <TitleBar />

        <div className="flex flex-1 overflow-hidden bg-card">
          {/* Left Sidebar Menu */}
          <aside className="w-64 backdrop-blur-xl flex flex-col bg-card">
            {/* Logo */}
            <div className="p-6">
              <div className="flex items-center gap-3">
                <QMLogo className="h-10 w-10 dark:invert" />
                <div>
                  <h1 className="text-xl font-bold text-foreground">
                    QMLauncher
                  </h1>
                  {appVersion && (
                    <span className="text-xs font-mono text-muted-foreground">
                      v{appVersion}
                    </span>
                  )}
                </div>
              </div>
            </div>
            
            {/* Navigation */}
            <nav className="flex-1 p-4 space-y-2 no-drag">
              <Button
                variant={activeTab === 'news' ? 'default' : 'ghost'}
                className="w-full justify-start no-drag"
                onClick={() => setActiveTab('news')}
              >
                <Newspaper className="w-5 h-5 mr-3" />
                {t('app.news')}
              </Button>
              
              <Button
                variant={activeTab === 'servers' ? 'default' : 'ghost'}
                className="w-full justify-start no-drag"
                onClick={() => setActiveTab('servers')}
              >
                <Server className="w-5 h-5 mr-3" />
                {t('app.servers')}
              </Button>
              
              <Button
                variant={activeTab === 'settings' ? 'default' : 'ghost'}
                className="w-full justify-start no-drag"
                onClick={() => setActiveTab('settings')}
              >
                <Settings className="w-5 h-5 mr-3" />
                {t('app.settings')}
              </Button>
            </nav>

            {/* User Menu */}
            {(appState === 'authenticated' || appState === 'offline') && authToken && (
              <NavUser authToken={authToken} onLogout={handleLogout} />
            )}
          </aside>

          {/* Main Content */}
          <main className="flex-1 overflow-hidden no-drag bg-background rounded-2xl m-2 ml-0">
            {appState === 'checking' && (
              <ServerConnectionCheck
                onServerAvailable={handleServerAvailable}
                onServerUnavailable={handleServerUnavailable}
              />
            )}
            
            {appState === 'login' && (
              <LoginForm onLoginSuccess={handleLoginSuccess} />
            )}
            
            {(appState === 'authenticated' || appState === 'offline') && (
              <motion.div
                key={activeTab}
                initial={{ opacity: 0, y: 10 }}
                animate={{ opacity: 1, y: 0 }}
                exit={{ opacity: 0, y: -10 }}
                transition={{ duration: 0.2 }}
                className="h-full overflow-auto no-drag"
              >
                {appState === 'offline' && (
                  <div className="bg-yellow-500/10 border-b border-yellow-500/20 p-3 text-center text-sm text-yellow-600 dark:text-yellow-400">
                    <span>⚠️ Сервер недоступен. Некоторые функции могут быть ограничены.</span>
                  </div>
                )}
                <Suspense fallback={
                  <div className="flex items-center justify-center h-full">
                    <Loader2 className="w-8 h-8 animate-spin text-primary" />
                  </div>
                }>
                  {activeTab === 'servers' && <ServersTab authToken={authToken} />}
                  {activeTab === 'news' && <NewsTab />}
                  {activeTab === 'settings' && <SettingsTab />}
                </Suspense>
              </motion.div>
            )}
          </main>
        </div>
        
        {/* Модальное окно ошибки - показывается поверх всего приложения */}
        <ErrorModal
          isOpen={showErrorModal}
          error={connectionError}
          onClose={handleCloseErrorModal}
          onRetry={handleRetryConnection}
        />
        
        {/* Диалог выбора режима работы (при первом запуске) */}
        {!isCheckingFirstLaunch && (
          <LauncherModeDialog
            open={showModeDialog}
            onModeSelected={handleModeSelected}
          />
        )}
        
        {/* Toast уведомления */}
        <Toaster />
      </div>
    </div>
  )
}

function App() {
  return (
    <ErrorBoundary>
      <QueryClientProvider client={queryClient}>
        <I18nProvider>
          <ThemeProvider>
            <AppContent />
          </ThemeProvider>
        </I18nProvider>
      </QueryClientProvider>
    </ErrorBoundary>
  )
}

export default App
