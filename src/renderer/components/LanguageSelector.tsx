import { Languages } from 'lucide-react'
import { useI18n } from '../contexts/I18nContext'
import { Button } from './ui/button'

export function LanguageSelector() {
  const { language, setLanguage, t } = useI18n()

  const handleLanguageToggle = async () => {
    const nextLanguage = language === 'ru' ? 'en' : 'ru'
    await setLanguage(nextLanguage)
  }

  return (
    <Button
      variant="ghost"
      size="icon"
      className="h-10 w-10 no-drag"
      onClick={handleLanguageToggle}
      title={t('language.switch')}
    >
      <Languages className="h-4 w-4" />
    </Button>
  )
}

