// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

import * as React from "react"
import { Menu } from "@base-ui/react/menu"
import { cn } from "@/lib/utils"

function DropdownMenu({ ...props }: Menu.Root.Props) {
  return <Menu.Root {...props} />
}

function DropdownMenuTrigger({ ...props }: Menu.Trigger.Props) {
  return <Menu.Trigger {...props} />
}

function DropdownMenuContent({
  className,
  align = "start",
  sideOffset = 4,
  ...props
}: Menu.Popup.Props & {
  align?: "start" | "end" | "center"
  sideOffset?: number
}) {
  return (
    <Menu.Portal>
      <Menu.Positioner align={align} side="bottom" sideOffset={sideOffset}>
        <Menu.Popup
          className={cn(
            "z-50 min-w-[10rem] overflow-hidden rounded-md border bg-popover p-1 text-popover-foreground shadow-md",
            "transition-[opacity,transform] duration-100 data-ending-style:opacity-0 data-starting-style:opacity-0",
            className
          )}
          {...props}
        />
      </Menu.Positioner>
    </Menu.Portal>
  )
}

function DropdownMenuItem({ className, ...props }: Menu.Item.Props) {
  return (
    <Menu.Item
      className={cn(
        "relative flex cursor-default select-none items-center gap-2 rounded-sm px-2 py-1.5 text-sm outline-none",
        "data-highlighted:bg-accent data-highlighted:text-accent-foreground",
        "data-disabled:pointer-events-none data-disabled:opacity-50",
        className
      )}
      {...props}
    />
  )
}

function DropdownMenuSeparator({ className, ...props }: React.ComponentProps<"div">) {
  return <div role="separator" className={cn("-mx-1 my-1 h-px bg-border", className)} {...props} />
}

function DropdownMenuLabel({ className, ...props }: React.ComponentProps<"div">) {
  return <div className={cn("px-2 py-1.5 text-xs font-semibold text-muted-foreground", className)} {...props} />
}

export {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuLabel,
}
