import { useState, useEffect } from 'react'
import { ServersTab } from './components/ServersTab'
import { NewsTab } from './components/NewsTab'
import { TitleBar } from './components/TitleBar'
import { ServerConnectionCheck } from './components/ServerConnectionCheck'
import { LoginForm } from './components/LoginForm'
import { ErrorModal } from './components/ErrorModal'
import { QMLogo } from './components/QMLogo'
import { Server, Newspaper } from 'lucide-react'
import { motion } from 'framer-motion'
import { Button } from './components/ui/button'
import { useI18n } from './contexts/I18nContext'
import './App.css'

type Tab = 'servers' | 'news'
type AppState = 'checking' | 'login' | 'authenticated' | 'error'

function App() {
  const { t } = useI18n()
  const [activeTab, setActiveTab] = useState<Tab>('news')
  const [appVersion, setAppVersion] = useState<string>('')
  const [appState, setAppState] = useState<AppState>('checking')
  const [authToken, setAuthToken] = useState<string | null>(null)
  const [connectionError, setConnectionError] = useState<string | null>(null)
  const [showErrorModal, setShowErrorModal] = useState(false)

  useEffect(() => {
    // Get version from build-time constant or Electron API
    const buildVersion = typeof __QM_LAUNCHER_VERSION__ !== 'undefined' ? __QM_LAUNCHER_VERSION__ : undefined
    if (buildVersion) {
      setAppVersion(buildVersion)
    } else if (window.electronAPI) {
      window.electronAPI.getAppVersion().then((version: string) => {
        setAppVersion(version)
      }).catch(() => {
        // Fallback if getAppVersion fails
      })
    }
    
    // Check if user is already authenticated
    if (window.electronAPI) {
      window.electronAPI.getAuthToken?.().then((token: string | null) => {
        if (token) {
          setAuthToken(token)
          setAppState('authenticated')
        }
      }).catch(() => {
        // No token, will show login form after server check
      })
    }
  }, [])

  const handleServerAvailable = () => {
    if (authToken) {
      setAppState('authenticated')
    } else {
      setAppState('login')
    }
  }

  const handleServerUnavailable = (error: string) => {
    setConnectionError(error)
    setShowErrorModal(true)
    setAppState('login') // Показываем форму авторизации за модальным окном
  }

  const handleRetryConnection = () => {
    setShowErrorModal(false)
    setConnectionError(null)
    setAppState('checking')
  }

  const handleCloseErrorModal = () => {
    setShowErrorModal(false)
  }

  const handleLoginSuccess = (token: string) => {
    setAuthToken(token)
    setAppState('authenticated')
  }

  return (
    <div className="h-screen flex flex-col overflow-hidden drag-region bg-transparent">
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
          </nav>
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
          
          {appState === 'authenticated' && (
            <motion.div
              key={activeTab}
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -10 }}
              transition={{ duration: 0.2 }}
              className="h-full overflow-auto no-drag"
            >
              {activeTab === 'servers' && <ServersTab authToken={authToken} />}
              {activeTab === 'news' && <NewsTab />}
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
      </div>
    </div>
  )
}

export default App
