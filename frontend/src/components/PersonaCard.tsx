import React from "react";
import { X } from "lucide-react";
import { Persona } from "../types";
import { ACTION_STYLES } from "../constants";

const ROUND_LABEL = ["Not reached", "Saw it first", "Got it from a share", "Recommended by the algorithm"];

// Detail card for the persona clicked on the spread graph.
export default function PersonaCard({ persona: p, onClose }: { persona: Persona; onClose: () => void }) {
  const style = ACTION_STYLES[p.action];
  return (
    <div className="absolute left-3 right-3 top-3 z-20 animate-fade-up rounded-xl border border-ink/10 bg-surface p-4 text-sm shadow-card sm:left-auto sm:w-80">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="font-display text-lg font-bold tracking-tight">{p.name}</div>
          <div className="text-xs text-ink/55">
            {p.age} · {p.gender} · {p.location}
          </div>
        </div>
        <button
          onClick={onClose}
          className="-m-1 rounded-md p-1 text-ink/45 hover:bg-ink/10 hover:text-ink"
          aria-label="Close"
        >
          <X className="h-4 w-4" />
        </button>
      </div>

      <div className="mt-3 flex flex-wrap items-center gap-2">
        <span className={`rounded-md px-2 py-0.5 text-xs font-medium ring-1 ring-inset ${style.badge}`}>{style.label}</span>
        {p.shown && <span className="text-xs text-ink/55">watched {p.watch}%</span>}
        <span className="text-xs text-ink/55">· {ROUND_LABEL[p.round] ?? ""}</span>
      </div>

      {p.why && p.shown && <p className="mt-3 rounded-lg bg-ink/5 p-3 font-serif text-base leading-snug text-ink/85">“{p.why}”</p>}

      <dl className="mt-3 grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 text-xs">
        <dt className="text-ink/50">Into</dt>
        <dd>{p.interests.join(", ")}</dd>
        <dt className="text-ink/50">Style</dt>
        <dd>{p.behavior}</dd>
        <dt className="text-ink/50">Followers</dt>
        <dd className="tabular-nums">{p.followers.toLocaleString()}</dd>
        <dt className="text-ink/50">Audience</dt>
        <dd>{p.inTarget ? "In your niche" : "Outside your niche"}</dd>
      </dl>
    </div>
  );
}
