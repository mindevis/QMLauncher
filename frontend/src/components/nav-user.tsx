import type { MouseEvent } from "react"
import { BadgeInfo, LogOut, MoreHorizontal, Settings2, Users } from "lucide-react"

import {
  Avatar,
  AvatarFallback,
  AvatarImage,
} from "./ui/avatar"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "./ui/dropdown-menu"
import {
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  useSidebar,
} from "./ui/sidebar"
import { useTranslate } from "../hooks/use-translate"

export function NavUser({
  user,
  accountClickable: _accountClickable,
  hasAccounts = false,
  onLogout,
  onAbout,
}: {
  user: {
    name: string
    email: string
    avatar: string
    avatarIsSkinTexture?: boolean
    isCloudPremium?: boolean
  }
  accountClickable?: boolean
  /** Hide “Log out” when there is no auth / game account to sign out from */
  hasAccounts?: boolean
  onLogout?: () => void
  onAbout?: () => void
}) {
  const { isMobile } = useSidebar()
  const gameAccountsText = useTranslate("ui.game_accounts")
  const settingsText = useTranslate("ui.settings")
  const aboutAppText = useTranslate("ui.about_app")
  const logoutText = useTranslate("ui.logout")
  const cloudPremiumLabel = useTranslate("ui.cloud_premium")

  const navigateToAccounts = (e?: React.MouseEvent) => {
    e?.preventDefault();
    e?.stopPropagation();
    window.location.hash = '#accounts';
    setTimeout(() => window.dispatchEvent(new Event('hashchange')), 0);
  }

  const navigateToGameAccounts = (e?: MouseEvent) => {
    e?.preventDefault();
    e?.stopPropagation();
    window.location.hash = '#game-accounts';
    setTimeout(() => window.dispatchEvent(new Event('hashchange')), 0);
  }

  const navigateToSettings = (e?: MouseEvent) => {
    e?.preventDefault();
    e?.stopPropagation();
    window.location.hash = '#settings';
    setTimeout(() => window.dispatchEvent(new Event('hashchange')), 0);
  }

  // Generate initials from username
  const getInitials = (name: string): string => {
    if (!name) return "U"
    const words = name.trim().split(/\s+/)
    if (words.length >= 2) {
      // If name has multiple words, use first letter of first two words
      return (words[0][0] + words[1][0]).toUpperCase()
    } else {
      // If single word, use first two letters
      return name.substring(0, 2).toUpperCase()
    }
  }

  const initials = getInitials(user.name)

  const skinTextureHeadStyle = user.avatarIsSkinTexture ? {
    backgroundImage: `url(${user.avatar})`,
    backgroundSize: "256px 256px",
    backgroundPosition: "-32px -32px",
    imageRendering: "pixelated" as const,
  } : undefined;

  const userBlockContent = (
    <>
      <Avatar className="h-8 w-8 rounded-lg overflow-hidden shrink-0">
        {user.avatarIsSkinTexture ? (
          <div className="size-full" style={skinTextureHeadStyle} />
        ) : (
          <AvatarImage src={user.avatar} alt={user.name} />
        )}
        {!user.avatarIsSkinTexture && <AvatarFallback className="rounded-lg">{initials}</AvatarFallback>}
      </Avatar>
      <div className="grid flex-1 text-left text-sm leading-tight min-w-0">
        <span className="flex min-w-0 items-center gap-1.5">
          <span className="truncate font-medium">{user.name}</span>
          {user.isCloudPremium ? (
            <span
              className="inline-flex max-w-full shrink-0 items-center rounded border border-amber-500/40 bg-amber-500/10 px-1 py-px text-[10px] font-medium leading-none text-amber-700 dark:text-amber-400"
              title={cloudPremiumLabel}
            >
              {cloudPremiumLabel}
            </span>
          ) : null}
        </span>
        <span className="text-muted-foreground truncate text-xs">
          {user.email}
        </span>
      </div>
    </>
  )

  return (
    <SidebarMenu>
      <SidebarMenuItem>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <SidebarMenuButton
              size="lg"
              className="data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground flex items-center gap-2"
            >
              <div className="flex flex-1 items-center gap-2 min-w-0">
                {userBlockContent}
              </div>
              <MoreHorizontal className="ml-auto size-4 shrink-0" />
            </SidebarMenuButton>
          </DropdownMenuTrigger>
          <DropdownMenuContent
            className="w-(--radix-dropdown-menu-trigger-width) min-w-56 rounded-lg"
            side={isMobile ? "bottom" : "right"}
            align="end"
            sideOffset={4}
          >
            <DropdownMenuItem
              onClick={navigateToAccounts}
              className="cursor-pointer focus:bg-sidebar-accent focus:text-sidebar-accent-foreground"
            >
              <div className="flex items-center gap-2 px-1 py-1.5 text-left text-sm w-full">
                <Avatar className="h-8 w-8 rounded-lg overflow-hidden shrink-0">
                  {user.avatarIsSkinTexture ? (
                    <div className="size-full" style={skinTextureHeadStyle} />
                  ) : (
                    <>
                      <AvatarImage src={user.avatar} alt={user.name} />
                      <AvatarFallback className="rounded-lg">{initials}</AvatarFallback>
                    </>
                  )}
                </Avatar>
                <div className="grid flex-1 text-left text-sm leading-tight min-w-0">
                  <span className="flex min-w-0 items-center gap-1.5">
                    <span className="truncate font-medium">{user.name}</span>
                    {user.isCloudPremium ? (
                      <span
                        className="inline-flex max-w-full shrink-0 items-center rounded border border-amber-500/40 bg-amber-500/10 px-1 py-px text-[10px] font-medium leading-none text-amber-700 dark:text-amber-400"
                        title={cloudPremiumLabel}
                      >
                        {cloudPremiumLabel}
                      </span>
                    ) : null}
                  </span>
                  <span className="text-muted-foreground truncate text-xs">
                    {user.email}
                  </span>
                </div>
              </div>
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuGroup>
              <DropdownMenuItem onClick={navigateToGameAccounts}>
                <div className="flex items-center gap-2">
                  <Users />
                  <span>{gameAccountsText}</span>
                </div>
              </DropdownMenuItem>
              <DropdownMenuItem onClick={navigateToSettings}>
                <div className="flex items-center gap-2">
                  <Settings2 />
                  <span>{settingsText}</span>
                </div>
              </DropdownMenuItem>
              <DropdownMenuItem
                onClick={(e) => {
                  e.preventDefault()
                  e.stopPropagation()
                  onAbout?.()
                }}
              >
                <div className="flex items-center gap-2">
                  <BadgeInfo />
                  <span>{aboutAppText}</span>
                </div>
              </DropdownMenuItem>
            </DropdownMenuGroup>
            {hasAccounts ? (
              <>
                <DropdownMenuSeparator />
                <DropdownMenuItem
                  className="cursor-pointer"
                  onClick={(e) => {
                    e.preventDefault()
                    e.stopPropagation()
                    onLogout?.()
                  }}
                >
                  <div className="flex items-center gap-2">
                    <LogOut />
                    <span>{logoutText}</span>
                  </div>
                </DropdownMenuItem>
              </>
            ) : null}
          </DropdownMenuContent>
        </DropdownMenu>
      </SidebarMenuItem>
    </SidebarMenu>
  )
}