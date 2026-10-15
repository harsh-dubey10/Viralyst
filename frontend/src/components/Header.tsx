import React from "react";
import { RotateCcw, Sun, Moon } from "lucide-react";
import { secondaryButton } from "./ui";

interface HeaderProps {
  darkMode: boolean;
  setDarkMode: (value: boolean) => void;
  canReset: boolean;
  resetAll: () => void;
}

// Glass square button (Shopify Editions .square-button).
const squareButton =
  "inline-flex h-[34px] w-[34px] items-center justify-center rounded-md border border-ink/15 bg-ink/5 text-ink/80 backdrop-blur transition duration-300 ease-spring hover:bg-ink hover:text-page focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent";

export default function Header({ darkMode, setDarkMode, canReset, resetAll }: HeaderProps) {
  return (
    <header className="sticky top-0 z-30 border-b border-ink/10 bg-page/60 backdrop-blur-xl">
      <div className="mx-auto flex h-[56px] max-w-[1680px] items-center justify-between px-4 sm:px-8">
        <a href="#top" className="flex animate-nav-in items-center gap-2.5" aria-label="Viralyst home">
          <span className="relative flex h-7 w-7 items-center justify-center rounded-full bg-ink">
            <span className="h-2.5 w-2.5 rounded-full bg-accent shadow-glow" />
          </span>
          <span className="font-display text-[17px] font-bold tracking-tight">Viralyst</span>
        </a>

        <nav className="flex items-center gap-2">
          <a href="#simulate" className={`${secondaryButton} hidden animate-nav-in sm:inline-flex`} style={{ animationDelay: "80ms" }}>
            Simulator
          </a>
          {canReset && (
            <button onClick={resetAll} className={`${secondaryButton} animate-nav-in`} style={{ animationDelay: "140ms" }}>
              <RotateCcw className="h-3.5 w-3.5" />
              <span className="hidden sm:inline">Start over</span>
            </button>
          )}
          <button
            onClick={() => setDarkMode(!darkMode)}
            className={`${squareButton} animate-nav-in`}
            style={{ animationDelay: "200ms" }}
            aria-label={darkMode ? "Switch to light mode" : "Switch to dark mode"}
            title={darkMode ? "Light mode" : "Dark mode"}
          >
            {darkMode ? <Sun className="h-4 w-4" /> : <Moon className="h-4 w-4" />}
          </button>
        </nav>
      </div>
    </header>
  );
}
