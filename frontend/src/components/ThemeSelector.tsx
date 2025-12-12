import { Sun, Moon } from 'lucide-react'
import { useTheme } from '../contexts/ThemeContext'
import { useI18n } from '../contexts/I18nContext'
import { Button } from './ui/button'

export function ThemeSelector() {
  const { currentTheme, setTheme } = useTheme()
  const { t } = useI18n()

  const toggleTheme = () => {
    const nextTheme = currentTheme.id === 'dark' ? 'light' : 'dark'
    setTheme(nextTheme)
  }

  return (
    <Button
      variant="ghost"
      size="icon"
      className="h-10 w-10 no-drag"
      onClick={toggleTheme}
      title={currentTheme.id === 'dark' ? t('theme.switchToLight') : t('theme.switchToDark')}
    >
      {currentTheme.id === 'dark' ? (
        <Sun className="h-4 w-4" />
      ) : (
        <Moon className="h-4 w-4" />
      )}
    </Button>
  )
}

