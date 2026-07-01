import type { PropsWithChildren } from "react";
import { useEffect } from "react";

import { useThemeStore } from "@/stores/theme-store";

export function ThemeProvider({ children }: PropsWithChildren) {
  const mode = useThemeStore((state) => state.mode);

  useEffect(() => {
    const root = document.documentElement;
    // Flip the theme with transitions momentarily disabled so all elements
    // switch at once instead of animating (which looked staggered/laggy).
    root.classList.add("theme-no-transitions");
    root.classList.toggle("dark", mode === "dark");
    root.style.colorScheme = mode;
    // Force a reflow so the class change is applied before transitions return.
    void root.offsetHeight;
    const frame = window.requestAnimationFrame(() => {
      root.classList.remove("theme-no-transitions");
    });
    return () => window.cancelAnimationFrame(frame);
  }, [mode]);

  return children;
}
