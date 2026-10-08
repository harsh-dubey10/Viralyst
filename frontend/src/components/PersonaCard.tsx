import React from "react";
import { X } from "lucide-react";
import { Persona } from "../types";
import { ACTION_STYLES } from "../constants";

const ROUND_LABEL = ["Not reached", "Saw it first", "Got it from a share", "Recommended by the algorithm"];

// Detail card for the persona clicked on the spread graph.
export default function PersonaCard({ persona: p, onClose }: { persona: Persona; onClose: () => void }) {
  const style = ACTION_STYLES[p.action];
  return (
    <div className="absolute left-3 right-3 top-3 z-20 rounded-xl border border-zinc-200 bg-white p-4 text-sm shadow-xl dark:border-zinc-700 dark:bg-zinc-900 sm:left-auto sm:w-80">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="font-semibold">{p.name}</div>
          <div className="text-xs text-zinc-500 dark:text-zinc-400">
            {p.age} · {p.gender} · {p.location}
          </div>
        </div>
        <button
          onClick={onClose}
          className="-m-1 rounded-md p-1 text-zinc-400 hover:bg-zinc-100 hover:text-zinc-700 dark:hover:bg-zinc-800 dark:hover:text-zinc-200"
          aria-label="Close"
        >
          <X className="h-4 w-4" />
        </button>
      </div>

      <div className="mt-3 flex flex-wrap items-center gap-2">
        <span className={`rounded-md px-2 py-0.5 text-xs font-medium ring-1 ring-inset ${style.badge}`}>{style.label}</span>
        {p.shown && <span className="text-xs text-zinc-500">watched {p.watch}%</span>}
        <span className="text-xs text-zinc-500">· {ROUND_LABEL[p.round] ?? ""}</span>
      </div>

      {p.why && p.shown && <p className="mt-3 rounded-lg bg-zinc-50 p-3 text-sm italic text-zinc-700 dark:bg-zinc-950 dark:text-zinc-300">"{p.why}"</p>}

      <dl className="mt-3 grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 text-xs">
        <dt className="text-zinc-500">Into</dt>
        <dd>{p.interests.join(", ")}</dd>
        <dt className="text-zinc-500">Style</dt>
        <dd>{p.behavior}</dd>
        <dt className="text-zinc-500">Followers</dt>
        <dd className="tabular-nums">{p.followers.toLocaleString()}</dd>
        <dt className="text-zinc-500">Audience</dt>
        <dd>{p.inTarget ? "In your niche" : "Outside your niche"}</dd>
      </dl>
    </div>
  );
}
