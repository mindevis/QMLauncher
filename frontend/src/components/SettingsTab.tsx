import { Settings as SettingsIcon } from 'lucide-react'
import { useI18n } from '../contexts/I18nContext'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from './ui/card'
import { useState, useEffect } from 'react'
import { wailsAPI } from '../bridge'

export function SettingsTab() {
  const { t } = useI18n()
  const [appVersion, setAppVersion] = useState<string>('')

  useEffect(() => {
    wailsAPI.getAppVersion().then(setAppVersion).catch(console.error)
  }, [])

  return (
    <div className="space-y-6 p-6">
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold flex items-center gap-2">
          <SettingsIcon className="w-6 h-6" />
          {t('settings.title') || 'Settings'}
        </h2>
      </div>

      {/* About */}
      <Card>
        <CardHeader>
          <CardTitle>{t('settings.about') || 'About'}</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-2">
            <CardDescription>
              {t('settings.aboutDescription') || 'QMLauncher - Minecraft Launcher'}
            </CardDescription>
            <CardDescription>
              {t('settings.version', {
                version: appVersion || (t('settings.versionLoading') || 'Loading...'),
              })}
            </CardDescription>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
