import { ACTION_STYLES } from "../constants";
import { Edge, Persona, Region } from "../types";

// Below this canvas width the two cohorts are stacked vertically instead of side by side.
export const NARROW_CANVAS = 640;

export function canvasSize(canvas: HTMLCanvasElement) {
  const W = canvas.offsetWidth || 900;
  return { W, H: W < NARROW_CANVAS ? 640 : 460 };
}


// Where each cohort's nodes live (below the cohort header badges).
export function cohortRegions(W: number, H: number): { inTarget: Region; outTarget: Region } {
  if (W < NARROW_CANVAS) {
    return {
      inTarget: { x0: 24, x1: W - 24, y0: 56, y1: H / 2 - 24 },
      outTarget: { x0: 24, x1: W - 24, y0: H / 2 + 56, y1: H - 28 },
    };
  }
  return {
    inTarget: { x0: 40, x1: W * 0.46, y0: 64, y1: H - 32 },
    outTarget: { x0: W * 0.56, x1: W - 40, y0: 64, y1: H - 32 },
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
  const minDist = W < NARROW_CANVAS ? 18 : 22;
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
  // Reallocating the backing store is the expensive part of a redraw (it happens on
  // every hover), so only do it when the size actually changed.
  if (canvas.width !== Math.round(W * dpr) || canvas.height !== Math.round(H * dpr)) {
    canvas.width = Math.round(W * dpr);
    canvas.height = Math.round(H * dpr);
    canvas.style.width = "100%"; // keep following the container so resizes are detected
    canvas.style.height = H + "px";
  }
  const ctx = canvas.getContext("2d");
  if (!ctx) return null;
  ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
  return { ctx, W, H };
}

// Theme colours for the canvas, matching the CSS tokens in index.css.
const SURFACE = { dark: "#171716", light: "#f7f7ee" };
const INK = { dark: "#f7f7ee", light: "#292919" };
const MUTED = { dark: "rgba(247,247,238,0.45)", light: "rgba(41,41,25,0.5)" };
const LINE = { dark: "rgba(247,247,238,0.08)", light: "rgba(41,41,25,0.1)" };
const theme = (isDark: boolean) => (isDark ? "dark" : "light");

function drawLabel(ctx: CanvasRenderingContext2D, x: number, y: number, title: string, hint: string, isDark: boolean) {
  ctx.font = '700 14px "Inter Tight", Inter, system-ui, sans-serif';
  ctx.fillStyle = INK[theme(isDark)];
  ctx.fillText(title, x, y);
  const w = ctx.measureText(title).width;
  ctx.font = "400 12px Inter, system-ui, sans-serif";
  ctx.fillStyle = MUTED[theme(isDark)];
  ctx.fillText(hint, x + w + 8, y);
}

// Background, cohort divider and cohort labels (shared by empty and result states).
export function drawBackdrop(ctx: CanvasRenderingContext2D, W: number, H: number, isDark: boolean) {
  const narrow = W < NARROW_CANVAS;
  ctx.fillStyle = SURFACE[theme(isDark)];
  ctx.fillRect(0, 0, W, H);

  ctx.save();
  ctx.beginPath();
  ctx.strokeStyle = LINE[theme(isDark)];
  ctx.lineWidth = 1;
  if (narrow) {
    ctx.moveTo(16, H / 2);
    ctx.lineTo(W - 16, H / 2);
  } else {
    ctx.moveTo(W * 0.51, 16);
    ctx.lineTo(W * 0.51, H - 16);
  }
  ctx.stroke();
  ctx.restore();

  drawLabel(ctx, narrow ? 20 : 40, 30, "Your audience", "in your niche", isDark);
  drawLabel(ctx, narrow ? 20 : W * 0.56, narrow ? H / 2 + 30 : 30, "Everyone else", "reached by shares & the algorithm", isDark);
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
        ctx.strokeStyle = ACTION_STYLES.share.fill[theme(isDark)];
        ctx.globalAlpha = isConnectedToSelected ? 1 : 0.6;
        ctx.lineWidth = isConnectedToSelected ? 2.4 : 1.4;
        ctx.setLineDash([]);
      } else {
        ctx.strokeStyle = isConnectedToSelected ? INK[theme(isDark)] : MUTED[theme(isDark)];
        ctx.globalAlpha = isConnectedToSelected ? 1 : 0.6;
        ctx.lineWidth = isConnectedToSelected ? 2 : 1;
        ctx.setLineDash([4, 4]);
      }
      ctx.stroke();
      ctx.setLineDash([]);
      ctx.globalAlpha = 1;
    });

  // Draw Personas
  personas.forEach((p) => {
    const isShownInRound = p.shown && p.round > 0 && p.round <= round;
    const isSelected = selectedId === p.id;
    const isHovered = hoveredId === p.id;

    const fill = isShownInRound ? ACTION_STYLES[p.action].fill[isDark ? "dark" : "light"] : ACTION_STYLES.never.fill[isDark ? "dark" : "light"];
    const isSharer = p.action === "share" && isShownInRound;
    const baseRadius = isSharer ? 10 : 7;
    const r = isSelected || isHovered ? baseRadius + 3 : baseRadius;

    if (isSelected || isHovered) {
      ctx.beginPath();
      ctx.arc(p.x, p.y, r + 5, 0, Math.PI * 2);
      ctx.fillStyle = isDark ? "rgba(247, 247, 238, 0.12)" : "rgba(41, 41, 25, 0.1)";
      ctx.fill();
    }

    ctx.beginPath();
    ctx.arc(p.x, p.y, r, 0, Math.PI * 2);
    ctx.fillStyle = fill;
    ctx.fill();

    // Node border
    ctx.strokeStyle = isSelected ? INK[theme(isDark)] : SURFACE[theme(isDark)];
    ctx.lineWidth = isSelected ? 2.5 : 1.5;
    ctx.stroke();
  });

  ctx.textAlign = "left";
  ctx.textBaseline = "alphabetic";
}
