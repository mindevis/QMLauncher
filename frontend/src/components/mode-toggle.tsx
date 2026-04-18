import { Switch } from "./ui/switch"
import { useTheme } from "../hooks/use-theme"

export function ModeToggle() {
  const { theme, setTheme } = useTheme()

  // checked = dark theme (explicit or system prefers dark)
  const isDark =
    theme === "dark" ||
    (theme === "system" && window.matchMedia("(prefers-color-scheme: dark)").matches)

  return (
    <Switch
      checked={isDark}
      onCheckedChange={(checked) => setTheme(checked ? "dark" : "light")}
      aria-label="Тема: светлая / тёмная"
    />
  )
}
