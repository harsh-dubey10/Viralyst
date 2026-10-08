import React from "react";
import { RotateCcw } from "lucide-react";
import { Action, Persona } from "../types";
import { ACTION_STYLES } from "../constants";
import PersonaCard from "./PersonaCard";
import { card, secondaryButton } from "./ui";

interface SpreadGraphProps {
  darkMode: boolean;
  currentRound: number;
  canvasRef: React.RefObject<HTMLCanvasElement | null>;
  handleCanvasClick: (e: React.MouseEvent<HTMLCanvasElement>) => void;
  handleCanvasMouseMove: (e: React.MouseEvent<HTMLCanvasElement>) => void;
  handleCanvasLeave: () => void;
  hoveredPersona: Persona | null;
  selectedPersona: Persona | null;
  closePersona: () => void;
  hasResult: boolean;
  replay: () => void;
  showRound: (round: number) => void;
}

const ROUNDS = [
  { n: 1, label: "First viewers" },
  { n: 2, label: "Shares" },
  { n: 3, label: "Algorithm" },
];

const LEGEND: Action[] = ["share", "comment", "like", "skip", "never"];

// The audience as a graph: one dot per simulated viewer, coloured by what they did.
export default function SpreadGraph({
  darkMode,
  currentRound,
  canvasRef,
  handleCanvasClick,
  handleCanvasMouseMove,
  handleCanvasLeave,
  hoveredPersona,
  selectedPersona,
  closePersona,
  hasResult,
  replay,
  showRound,
}: SpreadGraphProps) {
  return (
    <section className={`${card} overflow-hidden`}>
      <div className="flex flex-wrap items-center justify-between gap-3 px-5 py-4">
        <div>
          <h3 className="text-sm font-semibold">How it spread</h3>
          <p className="text-xs text-zinc-500 dark:text-zinc-400">Click any dot to see who they are and why they reacted.</p>
        </div>
        {hasResult && (
          <div className="flex items-center gap-2">
            <div className="inline-flex rounded-lg bg-zinc-100 p-0.5 dark:bg-zinc-800" role="tablist" aria-label="Spread round">
              {ROUNDS.map((r) => (
                <button
                  key={r.n}
                  role="tab"
                  aria-selected={currentRound === r.n}
                  onClick={() => showRound(r.n)}
                  className={`rounded-md px-2.5 py-1 text-xs font-medium transition ${
                    currentRound === r.n
                      ? "bg-white text-zinc-900 shadow-sm dark:bg-zinc-700 dark:text-white"
                      : "text-zinc-500 hover:text-zinc-800 dark:text-zinc-400 dark:hover:text-zinc-200"
                  }`}
                >
                  {r.n}. {r.label}
                </button>
              ))}
            </div>
            <button onClick={replay} className={secondaryButton} aria-label="Replay the spread">
              <RotateCcw className="h-3.5 w-3.5" />
              <span className="hidden sm:inline">Replay</span>
            </button>
          </div>
        )}
      </div>

      <div className="relative border-t border-zinc-100 dark:border-zinc-800">
        <canvas
          ref={canvasRef}
          onClick={handleCanvasClick}
          onMouseMove={handleCanvasMouseMove}
          onMouseLeave={handleCanvasLeave}
          className={`block w-full ${hasResult ? "cursor-pointer" : ""}`}
        />

        {hoveredPersona && !selectedPersona && (
          <div
            className="pointer-events-none absolute z-20 rounded-lg border border-zinc-200 bg-white px-3 py-2 text-xs shadow-lg dark:border-zinc-700 dark:bg-zinc-900"
            style={{
              left: Math.max(4, Math.min(hoveredPersona.x + 14, (canvasRef.current?.offsetWidth ?? 900) - 220)),
              top: Math.max(hoveredPersona.y - 50, 8),
            }}
          >
            <div className="flex items-center gap-1.5 font-medium">
              {hoveredPersona.name}
              <span className={`rounded px-1.5 py-px text-[10px] ring-1 ring-inset ${ACTION_STYLES[hoveredPersona.action].badge}`}>
                {ACTION_STYLES[hoveredPersona.action].label}
              </span>
            </div>
            <div className="text-zinc-500">{hoveredPersona.interests.join(", ")}</div>
          </div>
        )}

        {selectedPersona && <PersonaCard persona={selectedPersona} onClose={closePersona} />}
      </div>

      <div className="flex flex-wrap items-center gap-x-4 gap-y-1.5 border-t border-zinc-100 px-5 py-3 text-xs text-zinc-600 dark:border-zinc-800 dark:text-zinc-400">
        {LEGEND.map((a) => (
          <span key={a} className="inline-flex items-center gap-1.5">
            <span className="h-2.5 w-2.5 rounded-full" style={{ background: ACTION_STYLES[a].fill[darkMode ? "dark" : "light"] }} />
            {ACTION_STYLES[a].label}
          </span>
        ))}
        <span className="inline-flex items-center gap-1.5">
          <span className="w-4 border-t-2 border-emerald-500" /> Share
        </span>
        <span className="inline-flex items-center gap-1.5">
          <span className="w-4 border-t border-dashed border-zinc-400" /> Recommended
        </span>
      </div>
    </section>
  );
}
