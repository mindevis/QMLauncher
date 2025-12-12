"use client"

import {
  CircleCheck,
  Info,
  Loader2,
  XOctagon,
  TriangleAlert,
} from "lucide-react"
import { Toaster as Sonner, ToasterProps } from "sonner"
import { useTheme } from "../../contexts/ThemeContext"

const Toaster = ({ ...props }: ToasterProps) => {
  const { currentTheme } = useTheme()
  const theme = currentTheme.id === 'dark' ? 'dark' : 'light'

  return (
    <Sonner
      theme={theme as ToasterProps["theme"]}
      className="toaster group"
      position="top-center"
      icons={{
        success: <CircleCheck className="size-4" />,
        info: <Info className="size-4" />,
        warning: <TriangleAlert className="size-4" />,
        error: <XOctagon className="size-4" />,
        loading: <Loader2 className="size-4 animate-spin" />,
      }}
      style={
        {
          "--normal-bg": "var(--popover)",
          "--normal-text": "var(--popover-foreground)",
          "--normal-border": "var(--border)",
          "--border-radius": "var(--radius)",
        } as React.CSSProperties
      }
      {...props}
    />
  )
}

export { Toaster }

