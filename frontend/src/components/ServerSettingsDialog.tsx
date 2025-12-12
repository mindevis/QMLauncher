import { useState, useEffect } from 'react'
import { Server as ServerType } from '../shared/types'
import { Settings, FolderOpen, Monitor, HardDrive, Save, CheckCircle, XCircle, AlertCircle } from 'lucide-react'
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from './ui/dialog'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from './ui/card'
import { Button } from './ui/button'
import { Input } from './ui/input'
import { Label } from './ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from './ui/select'
import { useI18n } from '../contexts/I18nContext'
import { wailsAPI } from '../bridge'
import { toast } from 'sonner'

interface ServerSettingsDialogProps {
  server: ServerType | null
  isOpen: boolean
  onClose: () => void
}

export function ServerSettingsDialog({ server, isOpen, onClose }: ServerSettingsDialogProps) {
  const { t } = useI18n()
  const [settings, setSettings] = useState({
    minecraftPath: '',
    javaPath: '',
    jvmArgs: [] as string[],
    windowWidth: 1920,
    windowHeight: 1080,
    resolution: '1920x1080',
    customResolution: '',
    minMemory: 1024,
  })
  const [javaValidation, setJavaValidation] = useState<{ valid: boolean; version?: string; error?: string } | null>(null)
  const [isValidatingJava, setIsValidatingJava] = useState(false)
  const [isSaving, setIsSaving] = useState(false)
  const [resolutions, setResolutions] = useState<string[]>([])
  const [platform, setPlatform] = useState<string>('linux')

  useEffect(() => {
    if (isOpen && server) {
      loadSettings()
      loadResolutions()
      loadPlatform()
    }
  }, [isOpen, server])

  const loadPlatform = async () => {
    try {
      const plat = await wailsAPI.getPlatform()
      setPlatform(plat || 'linux')
    } catch (error) {
      console.error('Error loading platform:', error)
    }
  }

  const loadResolutions = async () => {
    try {
      const res = await wailsAPI.getScreenResolutions()
      if (res && Array.isArray(res) && res.length > 0) {
        setResolutions(res)
      } else {
        // Fallback to default resolutions
        setResolutions(['1920x1080', '1366x768', '1280x720', '1024x768'])
      }
    } catch (error) {
      console.error('Error loading resolutions:', error)
      setResolutions(['1920x1080', '1366x768', '1280x720', '1024x768'])
    }
  }

  const loadSettings = async () => {
    try {
      const config = await wailsAPI.getSettings()
      if (config) {
        let javaPath = config.javaPath || ''
        // Clean up old default paths
        if (javaPath === 'java' || (javaPath.includes('/.qmlauncher/java/bin/') && !javaPath.match(/\.qmlauncher\/[^\/]+\/java/))) {
          javaPath = ''
        }
        
        // Wails Settings structure may differ, adapt accordingly
        setSettings({
          minecraftPath: config.minecraftPath || '',
          javaPath: javaPath,
          jvmArgs: config.jvmArgs || [],
          windowWidth: 1920, // Default, Settings may not have these fields
          windowHeight: 1080,
          resolution: '1920x1080',
          customResolution: '',
          minMemory: config.minMemory || 1024,
        })
        
        if (javaPath) {
          validateJava(javaPath)
        }
      }
    } catch (error) {
      console.error('Error loading settings:', error)
    }
  }

  const validateJava = async (javaPath: string) => {
    if (!javaPath) {
      setJavaValidation(null)
      return
    }

    setIsValidatingJava(true)
    try {
      const result = await wailsAPI.validateJavaPath(javaPath)
      // Adapt result structure - Wails may return different format
      if (result.Valid !== undefined) {
        setJavaValidation({
          valid: result.Valid,
          version: result.Version,
          error: result.Error,
        })
      } else {
        // Fallback if structure is different
        setJavaValidation({ valid: false, error: 'Unknown validation result' })
      }
    } catch (error) {
      setJavaValidation({ valid: false, error: String(error) })
    } finally {
      setIsValidatingJava(false)
    }
  }

  const handleJavaPathChange = (value: string) => {
    setSettings({ ...settings, javaPath: value })
    setTimeout(() => validateJava(value), 500)
  }

  const handleJvmArgsChange = (value: string) => {
    const args = value
      .split(/\s+/)
      .filter(arg => arg.trim())
      .map(arg => arg.trim())
    setSettings({ ...settings, jvmArgs: args })
  }

  const handleResolutionChange = (value: string) => {
    if (value === 'custom') {
      setSettings({ ...settings, resolution: 'custom' })
    } else {
      const [width, height] = value.split('x').map(Number)
      setSettings({
        ...settings,
        resolution: value,
        windowWidth: width,
        windowHeight: height,
        customResolution: '',
      })
    }
  }

  const handleCustomResolutionChange = (value: string) => {
    const match = value.match(/^(\d+)x(\d+)$/)
    if (match) {
      const [, width, height] = match
      setSettings({
        ...settings,
        customResolution: value,
        windowWidth: parseInt(width),
        windowHeight: parseInt(height),
      })
    } else {
      setSettings({ ...settings, customResolution: value })
    }
  }

  const handleSave = async () => {
    setIsSaving(true)
    try {
      // Adapt settings to Wails Settings structure
      const wailsSettings = {
        apiBaseUrl: '', // Will be preserved from existing settings
        minecraftPath: settings.minecraftPath,
        javaPath: settings.javaPath,
        minMemory: settings.minMemory,
        maxMemory: 2048, // Default max memory
        jvmArgs: settings.jvmArgs,
      }
      
      // Get existing settings to preserve apiBaseUrl
      const existingSettings = await wailsAPI.getSettings()
      if (existingSettings && existingSettings.apiBaseUrl) {
        wailsSettings.apiBaseUrl = existingSettings.apiBaseUrl
      }
      
      await wailsAPI.saveSettings(wailsSettings)
      toast.success(t('settings.saved') || 'Настройки сохранены')
      onClose()
    } catch (error) {
      console.error('Error saving settings:', error)
      toast.error(t('settings.saveError') || 'Ошибка сохранения настроек', {
        description: error instanceof Error ? error.message : String(error)
      })
    } finally {
      setIsSaving(false)
    }
  }

  if (!server) return null

  const serverUuid = server.server_uuid || (server as any).embedded?.server_uuid || String(server.id)
  const javaExecutable = platform === 'windows' ? 'java.exe' : 'java'

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="max-w-4xl max-h-[90vh] overflow-y-auto hide-scrollbar">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Settings className="w-5 h-5" />
            {t('servers.clientSettings')}: {server.name}
          </DialogTitle>
          <DialogDescription className="flex items-center gap-4">
            <span>{t('servers.serverId') || 'Сервер ID:'} {server.id}</span>
            <span className="text-muted-foreground">|</span>
            <span>{t('servers.serverUuid') || 'Сервер UUID:'} {serverUuid}</span>
          </DialogDescription>
        </DialogHeader>
        
        <div className="space-y-6 mt-4">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            {/* Minecraft Settings */}
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <FolderOpen className="w-5 h-5" />
                  {t('settings.minecraftSettings') || 'Minecraft Settings'}
                </CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="space-y-2">
                  <Label>
                    {t('settings.minecraftPathLabel') || 'Path to Minecraft installation directory'}
                  </Label>
                  <Input
                    type="text"
                    value={settings.minecraftPath}
                    onChange={(e) => setSettings({ ...settings, minecraftPath: e.target.value })}
                    placeholder={`${t('settings.minecraftPathPlaceholder') || '~/.qmlauncher'}/${serverUuid}/minecraft`}
                  />
                </div>
              </CardContent>
            </Card>

            {/* Java Settings */}
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <Monitor className="w-5 h-5" />
                  {t('settings.javaSettings') || 'Java Settings'}
                </CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="space-y-2">
                  <Label>
                    {t('settings.javaPathLabel') || 'Path to Java installation directory'}
                  </Label>
                  <div className="flex items-center gap-2">
                    <Input
                      type="text"
                      value={settings.javaPath}
                      onChange={(e) => handleJavaPathChange(e.target.value)}
                      placeholder={`${t('settings.javaPathPlaceholder') || '~/.qmlauncher'}/${serverUuid}/java/bin/${javaExecutable}`}
                      className="flex-1"
                    />
                    {isValidatingJava && (
                      <div className="text-xs text-muted-foreground">{t('settings.javaValidating') || 'Validating...'}</div>
                    )}
                    {javaValidation && !isValidatingJava && (
                      <div className="flex items-center gap-1">
                        {javaValidation.valid ? (
                          <>
                            <CheckCircle className="w-4 h-4 text-green-500" />
                            {javaValidation.version && (
                              <span className="text-xs text-green-500">v{javaValidation.version}</span>
                            )}
                            <span className="text-xs text-green-500">{t('settings.javaValid') || 'Valid'}</span>
                          </>
                        ) : (
                          <>
                            <XCircle className="w-4 h-4 text-red-500" />
                            <span className="text-xs text-red-500" title={javaValidation.error}>
                              {t('settings.javaInvalid') || 'Invalid'}
                            </span>
                          </>
                        )}
                      </div>
                    )}
                  </div>
                  {javaValidation && !javaValidation.valid && javaValidation.error && (
                    <p className="text-xs text-destructive mt-1 flex items-center gap-1">
                      <AlertCircle className="w-3 h-3" />
                      {javaValidation.error}
                    </p>
                  )}
                </div>
                <div className="space-y-2">
                  <Label>
                    {t('settings.jvmArgs') || 'JVM Arguments (one per line or space-separated)'}
                  </Label>
                  <textarea
                    value={settings.jvmArgs.join(' ')}
                    onChange={(e) => handleJvmArgsChange(e.target.value)}
                    placeholder="-Xmx2G -XX:+UseG1GC ..."
                    className="flex min-h-[80px] w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-xs transition-colors placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50 focus-visible:border-ring disabled:cursor-not-allowed disabled:opacity-50 font-mono resize-none"
                    rows={4}
                  />
                </div>
              </CardContent>
            </Card>

            {/* Display Settings */}
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <HardDrive className="w-5 h-5" />
                  {t('settings.displaySettings') || 'Display Settings'}
                </CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="space-y-2">
                  <Label>
                    {t('settings.resolution') || 'Resolution'}
                  </Label>
                  <Select
                    value={settings.resolution}
                    onValueChange={handleResolutionChange}
                  >
                    <SelectTrigger className="w-full">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {resolutions.map((res) => (
                        <SelectItem key={res} value={res}>
                          {res}
                        </SelectItem>
                      ))}
                      <SelectItem value="custom">
                        {t('settings.customResolution') || 'Custom...'}
                      </SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                {settings.resolution === 'custom' && (
                  <div className="space-y-2">
                    <Label>
                      {t('settings.customResolutionInput') || 'Custom Resolution (e.g., 1920x1080)'}
                    </Label>
                    <Input
                      type="text"
                      value={settings.customResolution}
                      onChange={(e) => handleCustomResolutionChange(e.target.value)}
                      placeholder="1920x1080"
                    />
                  </div>
                )}
                <div className="grid grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Label>
                      {t('settings.windowWidth') || 'Width'}
                    </Label>
                    <Input
                      type="number"
                      value={settings.windowWidth}
                      onChange={(e) => setSettings({ ...settings, windowWidth: parseInt(e.target.value) || 1920 })}
                    />
                  </div>
                  <div className="space-y-2">
                    <Label>
                      {t('settings.windowHeight') || 'Height'}
                    </Label>
                    <Input
                      type="number"
                      value={settings.windowHeight}
                      onChange={(e) => setSettings({ ...settings, windowHeight: parseInt(e.target.value) || 1080 })}
                    />
                  </div>
                </div>
              </CardContent>
            </Card>

            {/* Memory Settings */}
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <Monitor className="w-5 h-5" />
                  {t('settings.memorySettings') || 'Memory Settings'}
                </CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="space-y-2">
                  <Label>
                    {t('settings.ram') || 'RAM (Xms)'}: {settings.minMemory} MB
                  </Label>
                  <input
                    type="range"
                    min="512"
                    max="8192"
                    step="256"
                    value={settings.minMemory}
                    onChange={(e) => setSettings({ ...settings, minMemory: parseInt(e.target.value) || 1024 })}
                    className="w-full"
                  />
                  <div className="flex justify-between text-xs text-muted-foreground mt-1">
                    <span>512 MB</span>
                    <span>8192 MB</span>
                  </div>
                </div>
              </CardContent>
            </Card>
          </div>

          <div className="flex items-center justify-end mt-6">
            <Button
              onClick={handleSave}
              disabled={isSaving}
              className="flex items-center gap-2"
            >
              <Save className="w-4 h-4" />
              {isSaving ? (t('settings.saving') || 'Saving...') : (t('settings.save') || 'Save')}
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}

