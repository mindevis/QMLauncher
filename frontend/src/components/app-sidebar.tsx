import * as React from "react"
import { FileText, Globe, Box, LayoutDashboard } from "lucide-react"

import { NavMain } from "./nav-main"
import { NavUser } from "./nav-user"
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from "./ui/sidebar"
import { useTranslate } from "../hooks/use-translate"
import { QMLogo } from "./qm-logo"

export function AppSidebar({
  currentAccount,
  accountClickable,
  disabled,
  hasAccounts,
  onLogout,
  onAbout,
  ...props
}: React.ComponentProps<typeof Sidebar> & {
  currentAccount?: { name: string; email: string; avatar?: string; avatarIsSkinTexture?: boolean; isCloudPremium?: boolean }
  accountClickable?: boolean
  disabled?: boolean
  hasAccounts?: boolean
  onLogout?: () => void
  onAbout?: () => void
}) {
  const [activeHash, setActiveHash] = React.useState(() => window.location.hash || '#news')
  const newsTitle = useTranslate("ui.news")
  const serversTitle = useTranslate("ui.servers")
  const instancesTitle = useTranslate("ui.instances")

  React.useEffect(() => {
    const handleHashChange = () => {
      setActiveHash(window.location.hash || '#news')
    }

    window.addEventListener('hashchange', handleHashChange)
    return () => window.removeEventListener('hashchange', handleHashChange)
  }, [])

  // Create navMain items with Lucide icons (shadcn style)
  const navMain = React.useMemo(() => [
    {
      title: "Dashboard",
      url: "#",
      icon: LayoutDashboard,
      hidden: true,
    },
    {
      title: newsTitle.replace(/^📰\s*/, ""),
      url: "#news",
      icon: FileText,
      isActive: activeHash === '#news' || (!activeHash || activeHash === '#'),
    },
    {
      title: serversTitle.replace(/^🌐\s*/, ""),
      url: "#servers",
      icon: Globe,
      isActive: activeHash === '#servers',
    },
    {
      title: instancesTitle.replace(/^📦\s*/, ""),
      url: "#instances",
      icon: Box,
      isActive:
        activeHash === '#instances' ||
        activeHash === '#instance' ||
        activeHash.startsWith('#instance-settings') ||
        activeHash.startsWith('#instance-resources'),
    },
  ], [activeHash, newsTitle, serversTitle, instancesTitle])

  // Use current account data; avatar from API (cloud) or skin (microsoft); when none — letter-based fallback
  const userData = {
    name: currentAccount?.name || "User",
    email: currentAccount?.email || "account@qmlauncher.local",
    avatar: currentAccount?.avatar || "",
    avatarIsSkinTexture: currentAccount?.avatarIsSkinTexture ?? false,
    isCloudPremium: currentAccount?.isCloudPremium ?? false,
  }

  return (
    <Sidebar collapsible="offcanvas" {...props}>
      <SidebarHeader>
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton
              asChild
              className="data-[slot=sidebar-menu-button]:!p-1.5"
            >
              <a href="#" className="flex items-center gap-2">
                <QMLogo className="size-5 shrink-0" />
                <span className="text-base font-semibold">QMLauncher</span>
              </a>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarHeader>
      <SidebarContent>
        <NavMain items={navMain} disabled={disabled} />
      </SidebarContent>
      <SidebarFooter>
        <NavUser
          user={userData}
          accountClickable={accountClickable}
          hasAccounts={hasAccounts}
          onLogout={onLogout}
          onAbout={onAbout}
        />
      </SidebarFooter>
    </Sidebar>
  )
}
