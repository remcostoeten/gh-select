"use client";

import { MoonIcon, SunIcon } from "@/shared/icons/icons";
import { useTheme } from "next-themes";

export function ThemeToggle() {
  const { setTheme, resolvedTheme } = useTheme();

  return (
    <button
      type="button"
      aria-label="Toggle theme"
      onClick={() => setTheme(resolvedTheme === "dark" ? "light" : "dark")}
      className="inline-flex size-8 items-center justify-center text-foreground/40 pressable hover:bg-foreground/5 hover:text-foreground/70 focus-visible:bg-foreground/8 focus-visible:text-foreground"
    >
      <SunIcon className="size-4 dark:hidden" />
      <MoonIcon className="hidden size-4 dark:block" />
    </button>
  );
}
