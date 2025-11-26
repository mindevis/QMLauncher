import { useState, useEffect } from 'react'
import { User, Plus, Trash2, Edit2 } from 'lucide-react'
import { cn } from '../lib/utils'
import { useI18n } from '../contexts/I18nContext'
import './ProfilesTab.css'

interface Profile {
  id: string
  name: string
  username: string
  serverId: number
  lastPlayed?: string
}

export function ProfilesTab() {
  const { t } = useI18n()
  const [profiles, setProfiles] = useState<Profile[]>([])
  const [selectedProfile, setSelectedProfile] = useState<Profile | null>(null)

  // Mock data for now
  const mockProfiles: Profile[] = [
    {
      id: '1',
      name: t('profiles.defaultProfile'),
      username: 'Player1',
      serverId: 1,
      lastPlayed: '2024-01-15',
    },
  ]

  useEffect(() => {
    // Load profiles from storage or API
    setProfiles(mockProfiles)
  }, [])

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold text-white flex items-center gap-2">
          <User className="w-6 h-6" />
          {t('profiles.title')}
        </h2>
        <button className="minecraft-button px-4 py-2 flex items-center gap-2">
          <Plus className="w-4 h-4" />
          {t('profiles.create')}
        </button>
      </div>

      {profiles.length === 0 ? (
        <div className="minecraft-card p-12 text-center">
          <User className="w-16 h-16 text-gray-500 mx-auto mb-4" />
          <p className="text-xl text-gray-300 mb-2">{t('profiles.noProfiles')}</p>
          <p className="text-gray-500 mb-4">{t('profiles.noProfilesDescription')}</p>
          <button className="minecraft-button">
            {t('profiles.createFirst')}
          </button>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {profiles.map((profile) => (
            <div
              key={profile.id}
              className={cn(
                "minecraft-card p-6 cursor-pointer transition-all hover:scale-105",
                selectedProfile?.id === profile.id && "ring-2 ring-minecraft-button-green"
              )}
              onClick={() => setSelectedProfile(profile)}
            >
              <div className="flex items-start justify-between mb-4">
                <div className="flex items-center gap-3">
                  <div className="w-12 h-12 rounded-lg bg-minecraft-button-green/20 flex items-center justify-center">
                    <User className="w-6 h-6 text-minecraft-button-green" />
                  </div>
                  <div>
                    <h3 className="text-lg font-bold text-white">{profile.name}</h3>
                    <p className="text-sm text-gray-400">{profile.username}</p>
                  </div>
                </div>
                <div className="flex gap-2">
                  <button className="p-2 hover:bg-gray-700/50 rounded transition-colors">
                    <Edit2 className="w-4 h-4 text-gray-400" />
                  </button>
                  <button className="p-2 hover:bg-red-600/20 rounded transition-colors">
                    <Trash2 className="w-4 h-4 text-red-400" />
                  </button>
                </div>
              </div>

              {profile.lastPlayed && (
                <p className="text-xs text-gray-500">
                  {t('profiles.lastPlayed', { date: new Date(profile.lastPlayed).toLocaleDateString() })}
                </p>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

