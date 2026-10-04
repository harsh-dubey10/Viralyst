import React from "react";
import { Flame, Check, Copy, Download, RotateCcw, Sun, Moon } from "lucide-react";
import { Analysis } from "../types";

interface HeaderProps {
  darkMode: boolean;
  setDarkMode: (value: boolean) => void;
  analysis: Analysis | null;
  simDone: boolean;
  copied: boolean;
  copySummary: () => void;
  exportJSON: () => void;
  resetAll: () => void;
  runId: string;
}

// Sticky top navbar: brand, theme toggle, report actions and run id.
export default function Header({
  darkMode,
  setDarkMode,
  analysis,
  simDone,
  copied,
  copySummary,
  exportJSON,
  resetAll,
  runId,
}: HeaderProps) {
  return (
    <header
      className={`sticky top-0 z-30 border-b px-4 sm:px-6 py-3 flex flex-wrap items-center justify-between gap-3 shadow-xs backdrop-blur-xl ${
        darkMode
          ? "border-slate-800/80 bg-slate-950/80"
          : "border-[#E7E1D2]/80 bg-white/85"
      }`}
    >
      <div className="flex items-center gap-3">
        <div className="flex items-center justify-center w-9 h-9 rounded-xl bg-gradient-to-tr from-indigo-600 to-cyan-500 text-white shadow-indigo-500/20 shadow-md">
          <Flame className="w-5 h-5" />
        </div>
        <div>
          <div className="flex items-center gap-2">
            <span className="font-extrabold text-base tracking-tight">
              Viralyst
            </span>
            <span
              className={`text-[10px] uppercase font-black tracking-wider px-2 py-0.5 rounded-full border ${
                darkMode
                  ? "bg-indigo-500/20 text-indigo-300 border-indigo-500/40"
                  : "bg-[#EFE9DC] text-[#4F3C21] border-[#DFD5C2]"
              }`}
            >
              {analysis ? (analysis.engine === "heuristic" ? "Offline model" : "LLM personas") : "v2.0"}
            </span>
          </div>
          <p
            className={`hidden sm:block text-[11px] font-medium ${
              darkMode ? "text-slate-400" : "text-[#706453]"
            }`}
          >
            Multi-Agent Synthetic Audience Viral Prediction Engine
          </p>
        </div>
      </div>

      <div className="flex flex-wrap items-center gap-2">
        {/* Dark / Light Mode Toggle */}
        <button
          onClick={() => setDarkMode(!darkMode)}
          className={`flex items-center gap-1.5 text-xs font-semibold px-3 py-1.5 rounded-xl border transition shadow-xs ${
            darkMode
              ? "bg-slate-900 hover:bg-slate-800 text-amber-300 border-slate-700"
              : "bg-[#FAF7F2] hover:bg-[#F0EBE0] text-slate-700 border-[#E2DAD0]"
          }`}
          title="Toggle Light / Dark Mode"
        >
          {darkMode ? (
            <>
              <Sun className="w-4 h-4 text-amber-400" />
              <span>Light</span>
            </>
          ) : (
            <>
              <Moon className="w-4 h-4 text-indigo-600" />
              <span>Dark</span>
            </>
          )}
        </button>

        {simDone && (
          <>
            <button
              onClick={copySummary}
              className={`flex items-center gap-1.5 text-xs px-3 py-1.5 rounded-xl border shadow-xs transition font-semibold ${
                darkMode
                  ? "bg-slate-900 hover:bg-slate-800 text-slate-200 border-slate-700"
                  : "bg-white hover:bg-slate-50 text-slate-700 border-slate-200"
              }`}
            >
              {copied ? (
                <Check className="w-3.5 h-3.5 text-emerald-500" />
              ) : (
                <Copy className="w-3.5 h-3.5 text-slate-400" />
              )}
              <span className="hidden sm:inline">{copied ? "Copied!" : "Copy Summary"}</span>
            </button>
            <button
              onClick={exportJSON}
              className={`flex items-center gap-1.5 text-xs px-3 py-1.5 rounded-xl border shadow-xs transition font-semibold ${
                darkMode
                  ? "bg-slate-900 hover:bg-slate-800 text-slate-200 border-slate-700"
                  : "bg-white hover:bg-slate-50 text-slate-700 border-slate-200"
              }`}
            >
              <Download className="w-3.5 h-3.5 text-slate-400" />
              <span className="hidden sm:inline">Export JSON</span>
            </button>
          </>
        )}
        <button
          onClick={resetAll}
          className={`flex items-center gap-1.5 text-xs transition px-2.5 py-1.5 rounded-xl font-semibold ${
            darkMode
              ? "text-slate-400 hover:text-white hover:bg-slate-900"
              : "text-slate-600 hover:text-slate-900 hover:bg-slate-100"
          }`}
        >
          <RotateCcw className="w-3.5 h-3.5 text-slate-400" />
          <span>Reset</span>
        </button>
        <div
          className={`hidden md:flex text-xs font-mono border px-3 py-1.5 rounded-xl items-center gap-2 font-semibold shadow-xs ${
            darkMode
              ? "bg-slate-900 border-slate-800 text-slate-300"
              : "bg-white border-[#E0D8CA] text-slate-700"
          }`}
        >
          <span className="w-2 h-2 rounded-full bg-emerald-500 animate-pulse" />
          <span>{runId || "Ready to Simulate"}</span>
        </div>
      </div>
    </header>
  );
}
