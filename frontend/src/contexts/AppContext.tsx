import { createContext, useContext, useState, ReactNode } from 'react'
import type { AppState, Theme, AppConfig } from '@/types'

// Default application configuration
const defaultConfig: AppConfig = {
  theme: 'system',
  window: {
    width: 1200,
    height: 800,
    center: true,
  },
  features: {
    autoUpdate: true,
    telemetry: false,
  },
}

// Application context interface
interface AppContextType {
  config: AppConfig
  setConfig: (config: Partial<AppConfig>) => void
  theme: Theme
  setTheme: (theme: Theme) => void
  state: AppState
  updateState: (updates: Partial<AppState>) => void
}

// Create context
const AppContext = createContext<AppContextType | undefined>(undefined)

// Context provider props
interface AppProviderProps {
  children: ReactNode
}

// Context provider component
export function AppProvider({ children }: AppProviderProps) {
  const [config, setConfigState] = useState<AppConfig>(defaultConfig)
  const [state, setState] = useState<AppState>({
    theme: config.theme,
    isLoading: false,
    error: null,
  })

  const setConfig = (updates: Partial<AppConfig>) => {
    setConfigState(prev => ({ ...prev, ...updates }))
  }

  const setTheme = (theme: Theme) => {
    setConfig({ theme })
    setState(prev => ({ ...prev, theme }))
  }

  const updateState = (updates: Partial<AppState>) => {
    setState(prev => ({ ...prev, ...updates }))
  }

  const value: AppContextType = {
    config,
    setConfig,
    theme: state.theme,
    setTheme,
    state,
    updateState,
  }

  return (
    <AppContext.Provider value={value}>
      {children}
    </AppContext.Provider>
  )
}

// Hook to use app context
export function useApp(): AppContextType {
  const context = useContext(AppContext)
  if (context === undefined) {
    throw new Error('useApp must be used within an AppProvider')
  }
  return context
}

// Export context for advanced usage
export { AppContext }
