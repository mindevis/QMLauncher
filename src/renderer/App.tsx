import { useState, useEffect } from 'react'
import { ServersTab } from './components/ServersTab'
import { ProfilesTab } from './components/ProfilesTab'
import { SettingsTab } from './components/SettingsTab'
import { Server, Settings, Bell, Search, HelpCircle, Newspaper } from 'lucide-react'
import './App.css'

type Tab = 'servers' | 'profiles' | 'settings' | 'news' | 'support'

function App() {
  const [activeTab, setActiveTab] = useState<Tab>('servers')
  const [appVersion, setAppVersion] = useState<string>('')

  useEffect(() => {
    if (window.electronAPI) {
      window.electronAPI.getAppVersion().then((version: string) => {
        setAppVersion(version)
      })
    }
  }, [])

  return (
    <div className="h-screen flex bg-gradient-to-br from-gray-900 via-gray-800 to-gray-900">
      {/* Left Sidebar */}
      <aside className="w-64 bg-gray-800/90 border-r-2 border-gray-700/50 flex flex-col">
        <div className="p-4 border-b border-gray-700/50">
          <h1 className="text-xl font-bold text-white">QMLauncher</h1>
          {appVersion && (
            <span className="text-xs text-gray-400">v{appVersion}</span>
          )}
        </div>
        
        <nav className="flex-1 p-4 space-y-2">
          <button
            className={`w-full flex items-center gap-3 px-4 py-3 rounded-lg font-semibold transition-all ${
              activeTab === 'servers'
                ? 'bg-blue-600 text-white shadow-lg'
                : 'text-gray-300 hover:bg-gray-700/50'
            }`}
            onClick={() => setActiveTab('servers')}
          >
            <Server className="w-5 h-5" />
            Сервера
          </button>
          
          <button
            className={`w-full flex items-center gap-3 px-4 py-3 rounded-lg font-semibold transition-all ${
              activeTab === 'profiles'
                ? 'bg-blue-600 text-white shadow-lg'
                : 'text-gray-300 hover:bg-gray-700/50'
            }`}
            onClick={() => setActiveTab('profiles')}
          >
            <Server className="w-5 h-5" />
            Сервера
          </button>
          
          <button
            className={`w-full flex items-center gap-3 px-4 py-3 rounded-lg font-semibold transition-all ${
              activeTab === 'settings'
                ? 'bg-blue-600 text-white shadow-lg'
                : 'text-gray-300 hover:bg-gray-700/50'
            }`}
            onClick={() => setActiveTab('settings')}
          >
            <Settings className="w-5 h-5" />
            Настройки
          </button>
          
          <button
            className={`w-full flex items-center gap-3 px-4 py-3 rounded-lg font-semibold transition-all ${
              activeTab === 'news'
                ? 'bg-blue-600 text-white shadow-lg'
                : 'text-gray-300 hover:bg-gray-700/50'
            }`}
            onClick={() => setActiveTab('news')}
          >
            <Newspaper className="w-5 h-5" />
            Новости
          </button>
          
          <button
            className={`w-full flex items-center gap-3 px-4 py-3 rounded-lg font-semibold transition-all ${
              activeTab === 'support'
                ? 'bg-blue-600 text-white shadow-lg'
                : 'text-gray-300 hover:bg-gray-700/50'
            }`}
            onClick={() => setActiveTab('support')}
          >
            <HelpCircle className="w-5 h-5" />
            Поддержка
          </button>
        </nav>
        
        <div className="p-4 border-t border-gray-700/50">
          <a
            href="#"
            className="text-gray-400 hover:text-white transition-colors text-sm flex items-center gap-2"
          >
            <span className="text-gray-500">&lt;</span> QMWeb
          </a>
        </div>
      </aside>

      {/* Main Content */}
      <div className="flex-1 flex flex-col">
        {/* Header */}
        <header className="bg-gray-800/90 border-b-2 border-gray-700/50 px-6 py-4">
          <div className="flex items-center justify-between">
            <h2 className="text-xl font-bold text-white">
              {activeTab === 'servers' && 'Сервера'}
              {activeTab === 'profiles' && 'Профили'}
              {activeTab === 'settings' && 'Настройки'}
              {activeTab === 'news' && 'Новости'}
              {activeTab === 'support' && 'Поддержка'}
            </h2>
            
            <div className="flex items-center gap-4">
              <div className="relative">
                <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 w-4 h-4 text-gray-400" />
                <input
                  type="text"
                  placeholder="Q Search"
                  className="bg-gray-700/50 border border-gray-600 rounded-lg pl-10 pr-4 py-2 text-white placeholder-gray-400 focus:outline-none focus:border-blue-500 w-64"
                />
              </div>
              
              <button className="p-2 text-gray-400 hover:text-white transition-colors">
                <Bell className="w-5 h-5" />
              </button>
              
              <button className="p-2 text-gray-400 hover:text-white transition-colors">
                <Settings className="w-5 h-5" />
              </button>
              
              <div className="w-8 h-8 rounded-full bg-gray-600 flex items-center justify-center">
                <span className="text-white text-xs font-bold">U</span>
              </div>
            </div>
          </div>
        </header>

        {/* Content Area */}
        <div className="flex-1 overflow-auto p-6">
          {activeTab === 'servers' && <ServersTab />}
          {activeTab === 'profiles' && <ProfilesTab />}
          {activeTab === 'settings' && <SettingsTab />}
          {activeTab === 'news' && (
            <div className="text-center py-12">
              <Newspaper className="w-16 h-16 text-gray-600 mx-auto mb-4" />
              <p className="text-xl text-gray-400">Новости</p>
            </div>
          )}
          {activeTab === 'support' && (
            <div className="text-center py-12">
              <HelpCircle className="w-16 h-16 text-gray-600 mx-auto mb-4" />
              <p className="text-xl text-gray-400">Поддержка</p>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

export default App
