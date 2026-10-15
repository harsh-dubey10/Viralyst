import React, { Suspense, lazy } from "react";
import { Globe2, Network, RotateCcw } from "lucide-react";
import { Action, Edge, Persona } from "../types";
import { ACTION_STYLES } from "../constants";
import PersonaCard from "./PersonaCard";
import { card, secondaryButton, sectionTitle } from "./ui";

// three.js is only downloaded when the globe is first shown.
const Globe3D = lazy(() => import("./Globe3D"));

export type GraphView = "globe" | "network";

interface SpreadGraphProps {
  darkMode: boolean;
  currentRound: number;
  canvasRef: React.RefObject<HTMLCanvasElement | null>;
  handleCanvasClick: (e: React.MouseEvent<HTMLCanvasElement>) => void;
  handleCanvasMouseMove: (e: React.MouseEvent<HTMLCanvasElement>) => void;
  handleCanvasLeave: () => void;
  hoveredPersona: Persona | null;
  selectedPersona: Persona | null;
  selectPersona: (p: Persona | null) => void;
  closePersona: () => void;
  hasResult: boolean;
  replay: () => void;
  showRound: (round: number) => void;
  view: GraphView;
  setView: (v: GraphView) => void;
  personas: Persona[];
  edges: Edge[];
  population: number;
}

const ROUNDS = [
  { n: 1, label: "First viewers" },
  { n: 2, label: "Shares" },
  { n: 3, label: "Algorithm" },
];

const LEGEND: Action[] = ["share", "comment", "like", "skip", "never"];

const VIEWS: { key: GraphView; label: string; Icon: typeof Globe2 }[] = [
  { key: "globe", label: "Globe", Icon: Globe2 },
  { key: "network", label: "Network", Icon: Network },
];

const segment = (active: boolean) =>
  `inline-flex items-center gap-1.5 rounded-full px-3 py-1 text-xs font-medium transition duration-300 ease-spring ${
    active ? "bg-ink text-page" : "text-ink/60 hover:text-ink"
  }`;

// The audience as a 3D globe or a 2D network: one dot per simulated viewer, coloured by what they did.
export default function SpreadGraph({
  darkMode,
  currentRound,
  canvasRef,
  handleCanvasClick,
  handleCanvasMouseMove,
  handleCanvasLeave,
  hoveredPersona,
  selectedPersona,
  selectPersona,
  closePersona,
  hasResult,
  replay,
  showRound,
  view,
  setView,
  personas,
  edges,
  population,
}: SpreadGraphProps) {
  const fallback = <div className="flex h-[460px] items-center justify-center text-sm text-ink/50 sm:h-[520px]">Loading 3D view…</div>;

  return (
    <section className={`${card} overflow-hidden`}>
      <div className="flex flex-wrap items-center justify-between gap-3 px-5 py-4 sm:px-6">
        <div>
          <div className={sectionTitle}>Spread</div>
          <h3 className="font-display text-xl font-bold tracking-tight">How it travelled</h3>
          <p className="text-xs text-ink/55">
            {view === "globe" ? "Drag to rotate. Click a viewer to see who they are and why they reacted." : "Click any dot to see who they are and why they reacted."}
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <div className="inline-flex rounded-full border border-ink/15 p-0.5" role="tablist" aria-label="View">
            {VIEWS.map(({ key, label, Icon }) => (
              <button key={key} role="tab" aria-selected={view === key} onClick={() => setView(key)} className={segment(view === key)}>
                <Icon className="h-3.5 w-3.5" /> {label}
              </button>
            ))}
          </div>
          {hasResult && (
            <>
              <div className="inline-flex rounded-full border border-ink/15 p-0.5" role="tablist" aria-label="Spread round">
                {ROUNDS.map((r) => (
                  <button key={r.n} role="tab" aria-selected={currentRound === r.n} onClick={() => showRound(r.n)} className={segment(currentRound === r.n)}>
                    {r.n}. {r.label}
                  </button>
                ))}
              </div>
              <button onClick={replay} className={secondaryButton} aria-label="Replay the spread">
                <RotateCcw className="h-3.5 w-3.5" />
                <span className="hidden sm:inline">Replay</span>
              </button>
            </>
          )}
        </div>
      </div>

      <div className="relative border-t border-ink/10">
        {view === "globe" && (
          <Suspense fallback={fallback}>
            <Globe3D
              dark={darkMode}
              personas={personas}
              edges={edges}
              placeholderCount={population}
              round={currentRound}
              selectedId={selectedPersona?.id ?? null}
              onSelect={selectPersona}
              onUnavailable={() => setView("network")}
            />
          </Suspense>
        )}

        {/* The 2D canvas stays mounted (hidden) so App can keep it laid out and drawn. */}
        <div className={view === "network" ? "" : "hidden"}>
          <canvas
            ref={canvasRef}
            onClick={handleCanvasClick}
            onMouseMove={handleCanvasMouseMove}
            onMouseLeave={handleCanvasLeave}
            className={`block w-full ${hasResult ? "cursor-pointer" : ""}`}
          />
          {hoveredPersona && !selectedPersona && (
            <div
              className="pointer-events-none absolute z-20 rounded-lg border border-ink/10 bg-surface px-3 py-2 text-xs shadow-card"
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
              <div className="text-ink/55">{hoveredPersona.interests.join(", ")}</div>
            </div>
          )}
        </div>

        {selectedPersona && <PersonaCard persona={selectedPersona} onClose={closePersona} />}
      </div>

      <div className="flex flex-wrap items-center gap-x-4 gap-y-1.5 border-t border-ink/10 px-5 py-3 text-xs text-ink/65 sm:px-6">
        {LEGEND.map((a) => (
          <span key={a} className="inline-flex items-center gap-1.5">
            <span className="h-2.5 w-2.5 rounded-full" style={{ background: ACTION_STYLES[a].fill[darkMode ? "dark" : "light"] }} />
            {ACTION_STYLES[a].label}
          </span>
        ))}
        <span className="inline-flex items-center gap-1.5">
          <span className="w-4 border-t-2" style={{ borderColor: ACTION_STYLES.share.fill[darkMode ? "dark" : "light"] }} /> Share
        </span>
        <span className="inline-flex items-center gap-1.5">
          <span className="w-4 border-t border-dashed border-ink/50" /> Recommended
        </span>
      </div>
    </section>
  );
}
