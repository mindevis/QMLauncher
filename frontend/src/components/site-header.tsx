import { Separator } from "./ui/separator"
import { SidebarTrigger } from "./ui/sidebar"

interface SiteHeaderProps {
  title: string
  /** Launcher semver label next to window controls, e.g. v1.0.10 */
  versionLabel?: string
}

export function SiteHeader({ title, versionLabel }: SiteHeaderProps) {
  return (
    <header className="flex h-(--header-height) shrink-0 items-center gap-2 border-b transition-[width,height] ease-linear group-has-data-[collapsible=icon]/sidebar-wrapper:h-(--header-height)">
      <div className="flex w-full min-w-0 items-center gap-1 px-4 lg:gap-2 lg:px-6">
        <SidebarTrigger className="-ml-1.5" />
        <Separator
          orientation="vertical"
          className="mx-2 data-[orientation=vertical]:h-4"
        />
        <h1 className="min-w-0 truncate text-base font-medium">{title}</h1>
        {versionLabel ? (
          <span
            className="ml-auto shrink-0 tabular-nums text-xs font-medium text-muted-foreground select-none"
            aria-hidden
          >
            {versionLabel}
          </span>
        ) : null}
      </div>
    </header>
  )
}