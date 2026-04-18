import { cn } from "@/lib/utils"

/** QM brand mark: `/logo.png` (same asset as QMAdmin); light glyph — use with invert for theme contrast. */
export function QMLogo({ className }: { className?: string }) {
  return (
    <img
      src="/logo.png"
      alt=""
      width={32}
      height={32}
      className={cn("size-8 shrink-0 rounded-[22%] object-cover invert dark:invert-0", className)}
    />
  )
}
