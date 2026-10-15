import React, { useEffect, useRef, useState } from "react";
import { AudienceGlobe } from "../three/AudienceGlobe";
import { webglAvailable } from "../lib/env";
import { actionColors, ACTION_STYLES } from "../constants";
import { Edge, Persona } from "../types";

interface Globe3DProps {
  dark: boolean;
  personas: Persona[];
  edges: Edge[];
  placeholderCount: number;
  round: number;
  selectedId: number | null;
  onSelect: (p: Persona | null) => void;
  onUnavailable: () => void;
}

// The audience as a draggable 3D globe (see three/AudienceGlobe). Falls back to the 2D
// view via onUnavailable when WebGL can't start.
export default function Globe3D({ dark, personas, edges, placeholderCount, round, selectedId, onSelect, onUnavailable }: Globe3DProps) {
  const hostRef = useRef<HTMLDivElement>(null);
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const globeRef = useRef<AudienceGlobe | null>(null);
  const personasRef = useRef(personas);
  personasRef.current = personas;
  const onSelectRef = useRef(onSelect);
  onSelectRef.current = onSelect;
  const [hover, setHover] = useState<{ p: Persona; x: number; y: number } | null>(null);

  useEffect(() => {
    if (!canvasRef.current || !hostRef.current) return;
    if (!webglAvailable()) {
      onUnavailable();
      return;
    }
    try {
      globeRef.current = new AudienceGlobe(canvasRef.current, hostRef.current, { dark, colors: actionColors(dark) }, {
        onSelect: (i) => onSelectRef.current(i === null ? null : personasRef.current[i] ?? null),
        onHover: (i, x, y) => {
          const p = i === null ? null : personasRef.current[i];
          setHover((h) => (p ? (h?.p === p ? h : { p, x, y }) : null));
        },
      });
    } catch {
      onUnavailable();
      return;
    }
    return () => {
      globeRef.current?.dispose();
      globeRef.current = null;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    const g = globeRef.current;
    if (!g) return;
    if (personas.length) g.setData(personas, edges);
    else g.setPlaceholder(placeholderCount);
  }, [personas, edges, placeholderCount]);

  useEffect(() => globeRef.current?.setRound(round), [round, personas]);
  useEffect(() => globeRef.current?.setTheme({ dark, colors: actionColors(dark) }), [dark]);
  useEffect(() => {
    const idx = selectedId === null ? -1 : personas.findIndex((p) => p.id === selectedId);
    globeRef.current?.setSelected(idx < 0 ? null : idx);
  }, [selectedId, personas]);

  return (
    <div ref={hostRef} className="relative h-[460px] w-full select-none sm:h-[520px]">
      <canvas ref={canvasRef} className="block h-full w-full cursor-grab touch-none active:cursor-grabbing" aria-label="3D audience globe — drag to rotate, click a viewer for details" />
      <div className="pointer-events-none absolute inset-x-0 bottom-3 flex justify-between px-5 text-[11px] font-medium uppercase tracking-[0.14em] text-ink/45">
        <span>← Your audience</span>
        <span>Everyone else →</span>
      </div>
      {hover && personas.length > 0 && selectedId === null && (
        <div
          className="pointer-events-none absolute z-20 rounded-lg border border-ink/10 bg-surface px-3 py-2 text-xs shadow-card"
          style={{ left: Math.max(4, Math.min(hover.x + 14, (hostRef.current?.clientWidth ?? 900) - 220)), top: Math.max(hover.y - 50, 8) }}
        >
          <div className="flex items-center gap-1.5 font-medium">
            {hover.p.name}
            <span className={`rounded px-1.5 py-px text-[10px] ring-1 ring-inset ${ACTION_STYLES[hover.p.action].badge}`}>{ACTION_STYLES[hover.p.action].label}</span>
          </div>
          <div className="text-ink/55">{hover.p.interests.join(", ")}</div>
        </div>
      )}
    </div>
  );
}
