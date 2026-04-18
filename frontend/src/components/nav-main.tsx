import type { MouseEvent } from "react"
import type { LucideIcon } from "lucide-react"

import {
  SidebarGroup,
  SidebarGroupContent,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from "./ui/sidebar"

export function NavMain({
  items,
  disabled,
}: {
  items: {
    title: string
    url: string
    icon?: LucideIcon
    hidden?: boolean
    isActive?: boolean
  }[]
  disabled?: boolean
}) {
  const handleClick = (e: MouseEvent<HTMLAnchorElement>) => {
    if (disabled) {
      e.preventDefault();
      e.stopPropagation();
    }
  };

  return (
    <SidebarGroup>
      <SidebarGroupContent className="flex flex-col gap-2">
        <SidebarMenu className="space-y-1">
          {items.map((item) => (
            <SidebarMenuItem key={item.title} className={item.hidden ? "hidden" : ""}>
              <SidebarMenuButton 
                asChild 
                tooltip={item.title} 
                isActive={item.isActive}
                disabled={disabled}
              >
                <a 
                  href={item.url} 
                  className={`flex items-center gap-2 ${disabled ? 'pointer-events-none opacity-50' : ''}`}
                  onClick={handleClick}
                >
                  {item.icon && <item.icon className="size-4 shrink-0" />}
                  <span>{item.title}</span>
                </a>
              </SidebarMenuButton>
            </SidebarMenuItem>
          ))}
        </SidebarMenu>
      </SidebarGroupContent>
    </SidebarGroup>
  )
}