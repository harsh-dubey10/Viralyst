import React from "react";
import { Action, ActionStyle, Persona } from "../types";

interface PersonaCardProps {
  selectedPersona: Persona;
  actionConfig: Record<Action, ActionStyle>;
  darkMode: boolean;
  onClose: () => void;
}

// Detail drawer for the persona clicked on the spread graph.
export default function PersonaCard({
  selectedPersona,
  actionConfig,
  darkMode,
  onClose,
}: PersonaCardProps) {
  return (
    <div
      className={`absolute top-4 left-4 right-4 sm:left-auto sm:w-80 border rounded-2xl p-4 z-20 backdrop-blur-2xl shadow-2xl animate-in fade-in zoom-in duration-150 ${
        darkMode
          ? "border-slate-700/90 bg-slate-950/95 text-slate-100"
          : "border-[#DFD6C4] bg-white/95 text-slate-900"
      }`}
    >
      <div className="flex items-center justify-between mb-2">
        <div className="flex items-center gap-2.5">
          <div className="w-8 h-8 rounded-xl bg-gradient-to-tr from-indigo-600 to-cyan-500 flex items-center justify-center font-black text-xs text-white shadow-sm">
            {selectedPersona.name.slice(0, 2).toUpperCase()}
          </div>
          <div>
            <div className="text-xs font-bold">
              {selectedPersona.name}
            </div>
            <div className="text-[10px] text-slate-500 font-medium">
              {selectedPersona.age}y · {selectedPersona.gender} ·{" "}
              {selectedPersona.location}
            </div>
          </div>
        </div>
        <button
          onClick={onClose}
          className="text-slate-400 hover:text-slate-700 dark:hover:text-white text-base leading-none p-1 font-bold"
        >
          ✕
        </button>
      </div>

      <div
        className={`space-y-2 text-xs pt-2 border-t ${
          darkMode ? "border-slate-800" : "border-[#EFE9DC]"
        }`}
      >
        <div className="flex justify-between">
          <span className="text-slate-500 font-medium">Followers:</span>
          <span className="font-mono font-bold">
            {selectedPersona.followers.toLocaleString()}
          </span>
        </div>
        <div className="flex justify-between">
          <span className="text-slate-500 font-medium">Behavior:</span>
          <span className="font-semibold">
            {selectedPersona.behavior}
          </span>
        </div>
        <div>
          <span className="text-slate-500 font-medium block mb-1">Interests:</span>
          <div className="flex flex-wrap gap-1">
            {selectedPersona.interests.map((int) => (
              <span
                key={int}
                className={`text-[10px] border px-2 py-0.5 rounded-md font-semibold ${
                  darkMode
                    ? "bg-slate-900 border-slate-800 text-indigo-300"
                    : "bg-[#F7F3EB] border-[#E8DFC9] text-[#5C4328]"
                }`}
              >
                {int}
              </span>
            ))}
          </div>
        </div>

        <div className="flex justify-between items-center pt-1.5">
          <span className="text-slate-500 font-medium">Decision:</span>
          <span
            className={`text-[10px] px-2 py-0.5 rounded-full font-bold border ${
              actionConfig[selectedPersona.action].badge
            }`}
          >
            {actionConfig[selectedPersona.action].label}
          </span>
        </div>

        <div>
          <div className="flex justify-between text-[11px] mb-1 font-medium">
            <span className="text-slate-500">Watch Retention:</span>
            <span className="font-bold text-indigo-600 dark:text-indigo-400 font-mono">
              {selectedPersona.watch}%
            </span>
          </div>
          <div
            className={`h-2 rounded-full overflow-hidden ${
              darkMode ? "bg-slate-800" : "bg-slate-200"
            }`}
          >
            <div
              className="h-full bg-gradient-to-r from-indigo-500 to-cyan-500 rounded-full"
              style={{ width: `${selectedPersona.watch}%` }}
            />
          </div>
        </div>

        {selectedPersona.why && (
          <div
            className={`pt-2 border-t ${
              darkMode ? "border-slate-800" : "border-[#EFE9DC]"
            }`}
          >
            <span className="text-[10px] font-bold text-slate-500 uppercase tracking-wider block mb-1">
              Persona reasoning:
            </span>
            <p
              className={`text-[11px] leading-relaxed italic p-2.5 rounded-xl border font-medium ${
                darkMode
                  ? "bg-slate-900/80 border-slate-800 text-slate-200"
                  : "bg-[#FAF7F2] border-[#E8DFC9] text-[#4A3C2A]"
              }`}
            >
              "{selectedPersona.why}"
            </p>
          </div>
        )}
      </div>
    </div>
  );
}
