import React from "react";
import { Play, Activity } from "lucide-react";
import { Action, ActionStyle, Persona } from "../types";
import PersonaCard from "./PersonaCard";

interface SpreadGraphProps {
  darkMode: boolean;
  spreadStatus: string;
  currentRound: number;
  canvasRef: React.RefObject<HTMLCanvasElement | null>;
  handleCanvasClick: (e: React.MouseEvent<HTMLCanvasElement>) => void;
  handleCanvasMouseMove: (e: React.MouseEvent<HTMLCanvasElement>) => void;
  hoveredPersona: Persona | null;
  selectedPersona: Persona | null;
  closePersona: () => void;
  actionConfig: Record<Action, ActionStyle>;
  simDone: boolean;
  replay: () => void;
  scrubRound: (e: React.MouseEvent<HTMLDivElement>) => void;
}

// Section 02: canvas spread visualizer with hover tooltip, persona drawer, replay bar and legend.
export default function SpreadGraph({
  darkMode,
  spreadStatus,
  currentRound,
  canvasRef,
  handleCanvasClick,
  handleCanvasMouseMove,
  hoveredPersona,
  selectedPersona,
  closePersona,
  actionConfig,
  simDone,
  replay,
  scrubRound,
}: SpreadGraphProps) {
  return (
    <section
      className={`border rounded-2xl shadow-xs overflow-hidden relative ${
        darkMode
          ? "border-slate-800/90 bg-slate-900/60 backdrop-blur-xl"
          : "border-[#E5DEC9] bg-white/95 backdrop-blur-md"
      }`}
    >
      <div
        className={`px-5 py-3 border-b flex items-center justify-between ${
          darkMode
            ? "border-slate-800 bg-slate-950/60"
            : "border-[#EAE3D2] bg-gradient-to-r from-[#FBF9F4] via-[#F4EFE6]/50 to-white"
        }`}
      >
        <div className="flex flex-wrap items-center gap-2 sm:gap-3">
          <span className="w-5 h-5 rounded-lg bg-indigo-600 text-white flex items-center justify-center text-[10px] font-black shadow-xs">
            02
          </span>
          <span className="text-xs font-bold tracking-wider">
            MULTI-ROUND VIRAL SPREAD GRAPH
          </span>
          <div
            className={`flex items-center gap-1.5 text-xs font-mono px-2.5 py-0.5 rounded-full font-bold border ${
              darkMode
                ? "bg-indigo-500/20 text-indigo-300 border-indigo-500/40"
                : "bg-[#EFE9DC] text-[#4A3924] border-[#DFD6C4]"
            }`}
          >
            <Activity className="w-3 h-3 text-indigo-500 animate-pulse" />
            <span>{spreadStatus}</span>
          </div>
        </div>

        <div className="flex items-center gap-2">
          <span className="text-[11px] font-mono font-bold text-slate-500">
            Round {currentRound || 1} / 3
          </span>
        </div>
      </div>

      <div className="relative">
        <canvas
          ref={canvasRef}
          onClick={handleCanvasClick}
          onMouseMove={handleCanvasMouseMove}
          className="w-full cursor-pointer block"
        />

        {/* Hover Tooltip */}
        {hoveredPersona && !selectedPersona && (
          <div
            className={`absolute pointer-events-none z-20 px-3.5 py-2 rounded-xl border shadow-xl text-xs backdrop-blur-md ${
              darkMode
                ? "bg-slate-950/95 border-slate-700 text-white"
                : "bg-white/95 border-[#E2DAD0] text-slate-900"
            }`}
            style={{
              left: Math.max(4, Math.min(hoveredPersona.x + 15, (canvasRef.current?.offsetWidth ?? 900) - 230)),
              top: Math.max(hoveredPersona.y - 45, 10),
            }}
          >
            <div className="font-bold flex items-center gap-1.5">
              <span>{hoveredPersona.name}</span>
              <span
                className={`text-[9px] px-1.5 py-0.2 rounded font-mono ${
                  actionConfig[hoveredPersona.action].badge
                }`}
              >
                {actionConfig[hoveredPersona.action].label}
              </span>
            </div>
            <div className="text-[10px] text-slate-500 font-medium">
              {hoveredPersona.location} · {hoveredPersona.followers.toLocaleString()} followers
            </div>
          </div>
        )}

        {/* Selected Persona Drawer / Card */}
        {selectedPersona && (
          <PersonaCard
            selectedPersona={selectedPersona}
            actionConfig={actionConfig}
            darkMode={darkMode}
            onClose={closePersona}
          />
        )}
      </div>

      {/* Replay Scrub Bar */}
      {simDone && (
        <div
          className={`flex flex-wrap items-center gap-3 sm:gap-4 px-4 sm:px-5 py-3 border-t ${
            darkMode
              ? "border-slate-800 bg-slate-950/80"
              : "border-[#EAE3D2] bg-[#FAF8F5]/90"
          }`}
        >
          <button
            onClick={replay}
            className="flex items-center gap-1.5 text-xs font-bold px-3.5 py-1.5 bg-indigo-600 hover:bg-indigo-700 rounded-xl text-white transition shadow-sm"
          >
            <Play className="w-3.5 h-3.5" /> REPLAY CASCADE
          </button>
          <div
            className={`flex-1 min-w-[120px] h-2 rounded-full cursor-pointer relative ${
              darkMode ? "bg-slate-800" : "bg-slate-200"
            }`}
            onClick={scrubRound}
          >
            <div
              className="h-full bg-gradient-to-r from-indigo-500 to-cyan-500 rounded-full transition-all"
              style={{ width: `${Math.round((currentRound / 3) * 100)}%` }}
            />
          </div>
          <span className="text-xs font-mono font-bold">
            Round {currentRound} / 3
          </span>
        </div>
      )}

      {/* Visual Legend */}
      <div
        className={`flex flex-wrap items-center justify-between gap-4 px-5 py-2.5 border-t text-xs font-semibold ${
          darkMode
            ? "border-slate-800 bg-slate-950/60 text-slate-400"
            : "border-[#EAE3D2] bg-white text-[#695B48]"
        }`}
      >
        <div className="flex flex-wrap items-center gap-x-5 gap-y-1">
          <div className="flex items-center gap-1.5">
            <div className="w-3 h-3 rounded-full bg-[#4F46E5]" />
            <span>Target Cohort</span>
          </div>
          <div className="flex items-center gap-1.5">
            <div className="w-3 h-3 rounded-full bg-[#F43F5E]" />
            <span>Out-of-Target Cohort</span>
          </div>
          <div className="flex items-center gap-1.5">
            <div className="w-3 h-3 rounded-full bg-[#059669]" />
            <span>Direct Sharer (Amplifier)</span>
          </div>
        </div>
        <div className="flex flex-wrap items-center gap-x-5 gap-y-1">
          <div className="flex items-center gap-1.5">
            <div className="w-4 h-0.5 bg-emerald-600" />
            <span>Direct Share (Green Curve)</span>
          </div>
          <div className="flex items-center gap-1.5">
            <div className="w-4 border-t border-dashed border-sky-500" />
            <span>Algo Push (Blue Dotted)</span>
          </div>
        </div>
      </div>
    </section>
  );
}
