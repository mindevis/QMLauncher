import * as React from "react"
import { cn } from "@/lib/utils"

export interface NativeSelectProps
  extends React.SelectHTMLAttributes<HTMLSelectElement> {}

const NativeSelect = React.forwardRef<HTMLSelectElement, NativeSelectProps>(
  ({ className, children, ...props }, ref) => {
    return (
      <select
        ref={ref}
        className={cn(
          "flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50",
          className
        )}
        {...props}
      >
        {children}
      </select>
    )
  }
)
NativeSelect.displayName = "NativeSelect"

export interface NativeSelectOptionProps
  extends React.OptionHTMLAttributes<HTMLOptionElement> {}

export const NativeSelectOption: React.FC<NativeSelectOptionProps> = ({
  children,
  ...props
}) => {
  return <option {...props}>{children}</option>
}

export { NativeSelect }