import { Edge, Persona, Region } from "../types";

// Below this canvas width the two cohorts are stacked vertically instead of side by side.
export const NARROW_CANVAS = 640;

export function canvasSize(canvas: HTMLCanvasElement) {
  const W = canvas.offsetWidth || 900;
  return { W, H: W < NARROW_CANVAS ? 720 : 540 };
}


// Where each cohort's nodes live (below the cohort header badges).
export function cohortRegions(W: number, H: number): { inTarget: Region; outTarget: Region } {
  if (W < NARROW_CANVAS) {
    return {
      inTarget: { x0: 24, x1: W - 24, y0: 62, y1: H / 2 - 24 },
      outTarget: { x0: 24, x1: W - 24, y0: H / 2 + 62, y1: H - 28 },
    };
  }
  return {
    inTarget: { x0: 50, x1: W * 0.46, y0: 78, y1: H - 40 },
    outTarget: { x0: W * 0.56, x1: W - 50, y0: 78, y1: H - 40 },
  };
}

function gridInto(ps: Persona[], r: Region) {
  const w = r.x1 - r.x0;
  const h = r.y1 - r.y0;
  const cols = Math.max(4, Math.ceil(Math.sqrt((ps.length * w) / Math.max(h, 1))));
  const numRows = Math.max(1, Math.ceil(ps.length / cols));
  ps.forEach((p, i) => {
    const col = i % cols;
    const row = Math.floor(i / cols);
    p.x = r.x0 + (col / Math.max(cols - 1, 1)) * w + (Math.random() - 0.5) * 16;
    p.y = r.y0 + (row / Math.max(numRows - 1, 1)) * h + (Math.random() - 0.5) * 14;
  });
}

export function layoutPersonas(ps: Persona[], W: number, H: number) {
  const regions = cohortRegions(W, H);
  const minDist = W < NARROW_CANVAS ? 20 : 26;
  gridInto(ps.filter((p) => p.inTarget), regions.inTarget);
  gridInto(ps.filter((p) => !p.inTarget), regions.outTarget);

  // Physics repulsion to spread overlapping nodes
  for (let iter = 0; iter < 45; iter++) {
    for (let i = 0; i < ps.length; i++) {
      for (let j = i + 1; j < ps.length; j++) {
        const dx = ps[j].x - ps[i].x;
        const dy = ps[j].y - ps[i].y;
        const dist = Math.sqrt(dx * dx + dy * dy);
        if (dist < minDist && dist > 0) {
          const force = (minDist - dist) / 2;
          const nx = (dx / dist) * force;
          const ny = (dy / dist) * force;
          ps[i].x -= nx;
          ps[i].y -= ny;
          ps[j].x += nx;
          ps[j].y += ny;
        }
      }
    }
  }

  // Clamp each node back into its own cohort region so nothing overlaps the headers or divider.
  ps.forEach((p) => {
    const r = p.inTarget ? regions.inTarget : regions.outTarget;
    p.x = Math.max(r.x0 - 12, Math.min(r.x1 + 12, p.x));
    p.y = Math.max(r.y0 - 8, Math.min(r.y1 + 8, p.y));
  });
}

export function setupCanvas(canvas: HTMLCanvasElement) {
  const dpr = window.devicePixelRatio || 1;
  const { W, H } = canvasSize(canvas);
  canvas.width = W * dpr;
  canvas.height = H * dpr;
  canvas.style.width = "100%"; // keep following the container so resizes are detected
  canvas.style.height = H + "px";
  const ctx = canvas.getContext("2d");
  if (!ctx) return null;
  ctx.scale(dpr, dpr);
  return { ctx, W, H };
}

function drawBadge(ctx: CanvasRenderingContext2D, x: number, y: number, text: string, fill: string, stroke: string, color: string) {
  ctx.font = "bold 10.5px Inter, sans-serif";
  const w = ctx.measureText(text).width + 20;
  ctx.fillStyle = fill;
  ctx.beginPath();
  ctx.roundRect(x, y, w, 26, 6);
  ctx.fill();
  ctx.strokeStyle = stroke;
  ctx.stroke();
  ctx.fillStyle = color;
  ctx.fillText(text, x + 10, y + 17);
}

// Background, grid, cohort divider and cohort header badges (shared by empty and result states).
export function drawBackdrop(ctx: CanvasRenderingContext2D, W: number, H: number, isDark: boolean) {
  const narrow = W < NARROW_CANVAS;
  const bgGrad = ctx.createLinearGradient(0, 0, W, H);
  bgGrad.addColorStop(0, isDark ? "#080C16" : "#FAF8F5");
  bgGrad.addColorStop(1, isDark ? "#0D1322" : "#F0F4F8");
  ctx.fillStyle = bgGrad;
  ctx.fillRect(0, 0, W, H);

  ctx.strokeStyle = isDark ? "rgba(255, 255, 255, 0.03)" : "rgba(148, 163, 184, 0.18)";
  ctx.lineWidth = 1;
  for (let x = 0; x < W; x += 36) {
    ctx.beginPath();
    ctx.moveTo(x, 0);
    ctx.lineTo(x, H);
    ctx.stroke();
  }
  for (let y = 0; y < H; y += 36) {
    ctx.beginPath();
    ctx.moveTo(0, y);
    ctx.lineTo(W, y);
    ctx.stroke();
  }

  ctx.save();
  ctx.beginPath();
  ctx.setLineDash([4, 5]);
  ctx.strokeStyle = isDark ? "rgba(99, 102, 241, 0.3)" : "rgba(99, 102, 241, 0.35)";
  ctx.lineWidth = 1.5;
  if (narrow) {
    ctx.moveTo(16, H / 2);
    ctx.lineTo(W - 16, H / 2);
  } else {
    ctx.moveTo(W * 0.51, 20);
    ctx.lineTo(W * 0.51, H - 20);
  }
  ctx.stroke();
  ctx.restore();

  ctx.save();
  drawBadge(ctx, narrow ? 12 : 36, 14, "TARGET DEMOGRAPHIC COHORT",
    isDark ? "rgba(99, 102, 241, 0.2)" : "rgba(99, 102, 241, 0.12)",
    isDark ? "rgba(99, 102, 241, 0.4)" : "rgba(99, 102, 241, 0.3)",
    isDark ? "#A5B4FC" : "#4338CA");
  drawBadge(ctx, narrow ? 12 : W * 0.56, narrow ? H / 2 + 14 : 14, "OUT-OF-TARGET COHORT",
    isDark ? "rgba(244, 63, 94, 0.2)" : "rgba(244, 63, 94, 0.12)",
    isDark ? "rgba(244, 63, 94, 0.4)" : "rgba(244, 63, 94, 0.3)",
    isDark ? "#FDA4AF" : "#BE123C");
  ctx.restore();
}

export function drawRound(
  canvas: HTMLCanvasElement,
  personas: Persona[],
  edges: Edge[],
  round: number,
  selectedId: number | null,
  hoveredId: number | null,
  isDark: boolean
) {
  const setup = setupCanvas(canvas);
  if (!setup) return;
  const { ctx, W, H } = setup;
  drawBackdrop(ctx, W, H, isDark);

  // Draw Smooth Curved Edges
  edges
    .filter((e) => e.round <= round)
    .forEach((e) => {
      const from = personas[e.from],
        to = personas[e.to];
      if (!from || !to) return;

      const isConnectedToSelected =
        selectedId !== null && (from.id === selectedId || to.id === selectedId);

      ctx.beginPath();
      ctx.moveTo(from.x, from.y);

      const midX = (from.x + to.x) / 2;
      const midY = (from.y + to.y) / 2 - 12;
      ctx.quadraticCurveTo(midX, midY, to.x, to.y);

      if (e.type === "direct") {
        ctx.strokeStyle = isConnectedToSelected
          ? "#10B981"
          : isDark
          ? "rgba(16, 185, 129, 0.75)"
          : "rgba(5, 150, 105, 0.7)";
        ctx.lineWidth = isConnectedToSelected ? 2.8 : 1.8;
        ctx.setLineDash([]);
      } else {
        ctx.strokeStyle = isConnectedToSelected
          ? "#38BDF8"
          : isDark
          ? "rgba(56, 189, 248, 0.65)"
          : "rgba(2, 132, 199, 0.6)";
        ctx.lineWidth = isConnectedToSelected ? 2.2 : 1.3;
        ctx.setLineDash([4, 4]);
      }
      ctx.stroke();
      ctx.setLineDash([]);
    });

  // Draw Personas
  personas.forEach((p) => {
    const isShownInRound = p.shown && p.round > 0 && p.round <= round;
    const isSelected = selectedId === p.id;
    const isHovered = hoveredId === p.id;

    let fill = p.inTarget ? "#4F46E5" : "#E11D48";
    if (p.action === "share" && isShownInRound) {
      fill = "#059669";
    } else if (!isShownInRound) {
      fill = isDark
        ? p.inTarget
          ? "#312E81"
          : "#881337"
        : p.inTarget
        ? "#C7D2FE"
        : "#FECDD3";
    }

    const baseRadius =
      p.action === "share" && isShownInRound ? 13 : p.inTarget ? 11 : 9;
    const r = isSelected || isHovered ? baseRadius + 3.5 : baseRadius;

    // Glowing halo ring for sharers, selected, or hovered nodes
    if ((p.action === "share" && isShownInRound) || isSelected || isHovered) {
      ctx.save();
      ctx.beginPath();
      ctx.arc(p.x, p.y, r + 7, 0, Math.PI * 2);
      ctx.fillStyle = isSelected
        ? "rgba(245, 158, 11, 0.3)"
        : isHovered
        ? "rgba(2, 132, 199, 0.25)"
        : "rgba(16, 185, 129, 0.25)";
      ctx.fill();
      ctx.restore();
    }

    // Node body
    ctx.beginPath();
    ctx.arc(p.x, p.y, r, 0, Math.PI * 2);
    ctx.fillStyle = fill;
    ctx.globalAlpha = isShownInRound ? 1.0 : isDark ? 0.4 : 0.65;
    ctx.fill();
    ctx.globalAlpha = 1.0;

    // Node border
    ctx.strokeStyle = isSelected
      ? "#F59E0B"
      : isHovered
      ? "#0284C7"
      : isShownInRound
      ? "#FFFFFF"
      : isDark
      ? "rgba(255, 255, 255, 0.2)"
      : "rgba(255, 255, 255, 0.85)";
    ctx.lineWidth = isSelected ? 3.5 : isHovered ? 2.5 : 2;
    ctx.stroke();

    // Node Icon
    if (isShownInRound && p.action !== "never" && p.action !== "skip") {
      ctx.fillStyle = "#FFFFFF";
      ctx.font = `bold ${r - 2}px sans-serif`;
      ctx.textAlign = "center";
      ctx.textBaseline = "middle";
      const icon = p.action === "like" ? "♥" : p.action === "share" ? "↗" : "◉";
      ctx.fillText(icon, p.x, p.y);
    }
  });

  ctx.textAlign = "left";
  ctx.textBaseline = "alphabetic";
}
