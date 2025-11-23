import { useState } from 'react'
import { Settings as SettingsIcon, Save, Monitor, HardDrive } from 'lucide-react'
import './SettingsTab.css'

export function SettingsTab() {
  const [settings, setSettings] = useState({
    javaPath: '',
    memory: 2048,
    windowWidth: 854,
    windowHeight: 480,
    fullscreen: false,
  })

  const handleSave = () => {
    // Save settings logic
    alert('Настройки сохранены!')
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold text-white flex items-center gap-2">
          <SettingsIcon className="w-6 h-6" />
          Настройки лаунчера
        </h2>
        <button
          onClick={handleSave}
          className="minecraft-button px-4 py-2 flex items-center gap-2"
        >
          <Save className="w-4 h-4" />
          Сохранить
        </button>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {/* Java Settings */}
        <div className="minecraft-card p-6">
          <div className="flex items-center gap-2 mb-4">
            <Monitor className="w-5 h-5 text-minecraft-button-green" />
            <h3 className="text-lg font-bold text-white">Настройки Java</h3>
          </div>
          <div className="space-y-4">
            <div>
              <label className="block text-sm text-gray-400 mb-2">
                Путь к Java
              </label>
              <input
                type="text"
                value={settings.javaPath}
                onChange={(e) => setSettings({ ...settings, javaPath: e.target.value })}
                placeholder="Автоопределение"
                className="minecraft-input w-full"
              />
            </div>
            <div>
              <label className="block text-sm text-gray-400 mb-2">
                Память (MB): {settings.memory}
              </label>
              <input
                type="range"
                min="1024"
                max="8192"
                step="256"
                value={settings.memory}
                onChange={(e) => setSettings({ ...settings, memory: parseInt(e.target.value) })}
                className="w-full"
              />
              <div className="flex justify-between text-xs text-gray-500 mt-1">
                <span>1 GB</span>
                <span>8 GB</span>
              </div>
            </div>
          </div>
        </div>

        {/* Display Settings */}
        <div className="minecraft-card p-6">
          <div className="flex items-center gap-2 mb-4">
            <HardDrive className="w-5 h-5 text-minecraft-button-green" />
            <h3 className="text-lg font-bold text-white">Настройки отображения</h3>
          </div>
          <div className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-sm text-gray-400 mb-2">
                  Ширина окна
                </label>
                <input
                  type="number"
                  value={settings.windowWidth}
                  onChange={(e) => setSettings({ ...settings, windowWidth: parseInt(e.target.value) })}
                  className="minecraft-input w-full"
                />
              </div>
              <div>
                <label className="block text-sm text-gray-400 mb-2">
                  Высота окна
                </label>
                <input
                  type="number"
                  value={settings.windowHeight}
                  onChange={(e) => setSettings({ ...settings, windowHeight: parseInt(e.target.value) })}
                  className="minecraft-input w-full"
                />
              </div>
            </div>
            <div>
              <label className="flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  checked={settings.fullscreen}
                  onChange={(e) => setSettings({ ...settings, fullscreen: e.target.checked })}
                  className="w-4 h-4 rounded"
                />
                <span className="text-sm text-gray-400">Полноэкранный режим</span>
              </label>
            </div>
          </div>
        </div>
      </div>

      {/* About */}
      <div className="minecraft-card p-6">
        <h3 className="text-lg font-bold text-white mb-4">О лаунчере</h3>
        <div className="space-y-2 text-sm text-gray-400">
          <p>QMLauncher - современный лаунчер для Minecraft серверов</p>
          <p>Версия: {window.electronAPI ? 'Загрузка...' : 'N/A'}</p>
        </div>
      </div>
    </div>
  )
}
