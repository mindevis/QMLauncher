import { createContext, useContext, useState, useEffect, ReactNode } from 'react'

export interface ThemeColors {
  background: string
  backgroundSecondary: string
  backgroundTertiary: string
  foreground: string
  foregroundSecondary: string
  border: string
  borderSecondary: string
  primary: string
  primaryHover: string
  secondary: string
  secondaryHover: string
  accent: string
  accentHover: string
  text: string
  textSecondary: string
  textMuted: string
  success: string
  error: string
  warning: string
  info: string
}

export interface Theme {
  id: string
  name: string
  displayName: string
  colors: ThemeColors
  isCustom?: boolean
}

const defaultThemes: Theme[] = [
  {
    id: 'dark',
    name: 'dark',
    displayName: 'Темная',
    colors: {
      background: '#252525', // oklch(0.145 0 0)
      backgroundSecondary: '#343434', // oklch(0.205 0 0)
      backgroundTertiary: '#444444', // oklch(0.269 0 0)
      foreground: '#fafafa', // oklch(0.985 0 0)
      foregroundSecondary: '#f5f5f5',
      border: 'rgba(255, 255, 255, 0.1)', // oklch(1 0 0 / 10%)
      borderSecondary: 'rgba(255, 255, 255, 0.15)',
      primary: '#ebebeb', // oklch(0.922 0 0)
      primaryHover: '#d0d0d0',
      secondary: '#444444', // oklch(0.269 0 0)
      secondaryHover: '#555555',
      accent: '#444444', // oklch(0.269 0 0)
      accentHover: '#555555',
      text: '#fafafa', // oklch(0.985 0 0)
      textSecondary: '#e0e0e0',
      textMuted: '#b5b5b5', // oklch(0.708 0 0)
      success: '#10b981',
      error: '#ef4444',
      warning: '#f59e0b',
      info: '#3b82f6'
    }
  },
  {
    id: 'light',
    name: 'light',
    displayName: 'Светлая',
    colors: {
      background: '#ffffff', // oklch(1 0 0)
      backgroundSecondary: '#f7f7f7', // oklch(0.97 0 0)
      backgroundTertiary: '#f5f5f5',
      foreground: '#252525', // oklch(0.145 0 0)
      foregroundSecondary: '#1a1a1a',
      border: '#ebebeb', // oklch(0.922 0 0)
      borderSecondary: '#d0d0d0',
      primary: '#343434', // oklch(0.205 0 0)
      primaryHover: '#252525',
      secondary: '#f7f7f7', // oklch(0.97 0 0)
      secondaryHover: '#e8e8e8',
      accent: '#f7f7f7', // oklch(0.97 0 0)
      accentHover: '#e8e8e8',
      text: '#252525', // oklch(0.145 0 0)
      textSecondary: '#404040',
      textMuted: '#8e8e8e', // oklch(0.556 0 0)
      success: '#10b981',
      error: '#ef4444',
      warning: '#f59e0b',
      info: '#3b82f6'
    }
  }
]

interface ThemeContextType {
  currentTheme: Theme
  themes: Theme[]
  setTheme: (themeId: string) => void
}

const ThemeContext = createContext<ThemeContextType | undefined>(undefined)

export function ThemeProvider({ children }: { children: ReactNode }) {
  const [themes] = useState<Theme[]>(defaultThemes)
  const [currentTheme, setCurrentTheme] = useState<Theme>(defaultThemes[0])

  // Load theme from storage on mount
  useEffect(() => {
    loadThemeFromStorage()
  }, [])

  // Apply theme to CSS variables
  useEffect(() => {
    applyTheme(currentTheme)
    saveThemeToStorage(currentTheme.id)
  }, [currentTheme])

  const loadThemeFromStorage = async () => {
    try {
      if (window.electronAPI) {
        const config = await window.electronAPI.getLauncherConfig()
        if (config?.themeId) {
          const theme = themes.find(t => t.id === config.themeId)
          if (theme) {
            setCurrentTheme(theme)
            return
          }
        }
      }
    } catch (error) {
      console.error('Error loading theme from storage:', error)
    }
  }

  const saveThemeToStorage = async (themeId: string) => {
    try {
      if (window.electronAPI) {
        const config = await window.electronAPI.getLauncherConfig() || {}
        await window.electronAPI.saveLauncherConfig({
          ...config,
          themeId
        })
      }
    } catch (error) {
      console.error('Error saving theme to storage:', error)
    }
  }

  const applyTheme = (theme: Theme) => {
    const root = document.documentElement
    
    // Просто переключаем класс dark, CSS переменные уже определены в index.css
    if (theme.id === 'dark') {
      root.classList.add('dark')
    } else {
      root.classList.remove('dark')
    }
  }

  const setTheme = (themeId: string) => {
    const theme = themes.find(t => t.id === themeId)
    if (theme) {
      setCurrentTheme(theme)
    }
  }

  return (
    <ThemeContext.Provider
      value={{
        currentTheme,
        themes,
        setTheme
      }}
    >
      {children}
    </ThemeContext.Provider>
  )
}

export function useTheme() {
  const context = useContext(ThemeContext)
  if (context === undefined) {
    throw new Error('useTheme must be used within a ThemeProvider')
  }
  return context
}

