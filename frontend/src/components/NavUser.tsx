import { useState, useEffect } from 'react'
import { MoreVertical, User, LogOut } from 'lucide-react'
import {
  Avatar,
  AvatarFallback,
  AvatarImage,
} from './ui/avatar'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from './ui/dropdown-menu'
import { Button } from './ui/button'
import { Skeleton } from './ui/skeleton'
import { useI18n } from '../contexts/I18nContext'
import { api } from '../utils/api-client'

interface NavUserProps {
  authToken: string | null
  onLogout: () => void
}

interface UserInfo {
  username: string
  email: string
}

export function NavUser({ authToken, onLogout }: NavUserProps) {
  const { t } = useI18n()
  const [userInfo, setUserInfo] = useState<UserInfo | null>(null)
  const [isLoading, setIsLoading] = useState(true)

  useEffect(() => {
    if (authToken) {
      loadUserInfo()
    } else {
      setIsLoading(false)
    }
  }, [authToken])

  const loadUserInfo = async () => {
    if (!authToken) return
    
    try {
      const response = await api.get<{ username: string; email: string }>('/auth/me', {
        authToken,
      })
      
      if (response.ok && response.data) {
        setUserInfo({
          username: response.data.username || response.data.email.split('@')[0],
          email: response.data.email || '',
        })
      }
    } catch (error) {
      console.error('Error loading user info:', error)
    } finally {
      setIsLoading(false)
    }
  }

  const handleLogout = () => {
    localStorage.removeItem('qmlauncher_auth_token')
    onLogout()
  }

  // Generate initials from username
  const getInitials = (name: string): string => {
    if (!name) return "U"
    const words = name.trim().split(/\s+/)
    if (words.length >= 2) {
      return (words[0][0] + words[1][0]).toUpperCase()
    } else {
      return name.substring(0, 2).toUpperCase()
    }
  }

  if (isLoading || !userInfo) {
    return (
      <div className="p-4 border-t">
        <div className="flex items-center gap-3">
          <Skeleton className="h-8 w-8 rounded-lg" />
          <div className="flex-1 space-y-2">
            <Skeleton className="h-4 w-24" />
            <Skeleton className="h-3 w-32" />
          </div>
        </div>
      </div>
    )
  }

  const initials = getInitials(userInfo.username)

  return (
    <div className="p-4 border-t">
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button
            variant="ghost"
            className="w-full justify-start h-auto p-3 hover:bg-accent"
          >
            <Avatar className="h-8 w-8 rounded-lg">
              <AvatarImage src="" alt={userInfo.username} />
              <AvatarFallback className="rounded-lg">{initials}</AvatarFallback>
            </Avatar>
            <div className="flex-1 text-left ml-3 min-w-0">
              <div className="text-sm font-medium truncate">{userInfo.username}</div>
              <div className="text-xs text-muted-foreground truncate">{userInfo.email}</div>
            </div>
            <MoreVertical className="ml-auto h-4 w-4" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent
          className="w-[var(--radix-dropdown-menu-trigger-width)] min-w-56 rounded-lg"
          side="right"
          align="end"
          sideOffset={4}
        >
          <DropdownMenuLabel className="p-0 font-normal">
            <div className="flex items-center gap-2 px-1 py-1.5 text-left text-sm">
              <Avatar className="h-8 w-8 rounded-lg">
                <AvatarImage src="" alt={userInfo.username} />
                <AvatarFallback className="rounded-lg">{initials}</AvatarFallback>
              </Avatar>
              <div className="flex-1 text-left text-sm leading-tight min-w-0">
                <div className="truncate font-medium">{userInfo.username}</div>
                <div className="truncate text-xs text-muted-foreground">{userInfo.email}</div>
              </div>
            </div>
          </DropdownMenuLabel>
          <DropdownMenuSeparator />
          <DropdownMenuGroup>
            <DropdownMenuItem
              onClick={() => {
                const event = new CustomEvent('navigate', { detail: 'settings' })
                window.dispatchEvent(event)
              }}
            >
              <User className="mr-2 h-4 w-4" />
              {t('app.account') || 'Account'}
            </DropdownMenuItem>
          </DropdownMenuGroup>
          <DropdownMenuSeparator />
          <DropdownMenuItem onClick={handleLogout}>
            <LogOut className="mr-2 h-4 w-4" />
            {t('login.logout') || 'Logout'}
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  )
}

