import React from "react";
import { Flame, RotateCcw, Sun, Moon } from "lucide-react";

interface HeaderProps {
  darkMode: boolean;
  setDarkMode: (value: boolean) => void;
  canReset: boolean;
  resetAll: () => void;
}

const iconButton =
  "inline-flex h-9 items-center gap-1.5 rounded-lg px-2.5 text-sm font-medium text-zinc-600 transition hover:bg-zinc-100 hover:text-zinc-900 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-indigo-500 dark:text-zinc-400 dark:hover:bg-zinc-800 dark:hover:text-zinc-100";

export default function Header({ darkMode, setDarkMode, canReset, resetAll }: HeaderProps) {
  return (
    <header className="sticky top-0 z-30 border-b border-zinc-200 bg-white/80 backdrop-blur dark:border-zinc-800 dark:bg-zinc-950/80">
      <div className="mx-auto flex h-14 max-w-7xl items-center justify-between px-4 sm:px-6">
        <div className="flex items-center gap-2.5">
          <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-indigo-600 text-white">
            <Flame className="h-4 w-4" />
          </div>
          <span className="text-[15px] font-semibold tracking-tight">Viralyst</span>
          <span className="hidden text-sm text-zinc-500 sm:inline">Test your hook on a synthetic audience before you post</span>
        </div>

        <div className="flex items-center gap-1">
          {canReset && (
            <button onClick={resetAll} className={iconButton}>
              <RotateCcw className="h-4 w-4" />
              <span className="hidden sm:inline">Start over</span>
            </button>
          )}
          <button
            onClick={() => setDarkMode(!darkMode)}
            className={iconButton}
            aria-label={darkMode ? "Switch to light mode" : "Switch to dark mode"}
            title={darkMode ? "Light mode" : "Dark mode"}
          >
            {darkMode ? <Sun className="h-4 w-4" /> : <Moon className="h-4 w-4" />}
          </button>
        </div>
      </div>
    </header>
  );
}
