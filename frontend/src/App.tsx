import React, { useState, useRef, useEffect, useCallback, useMemo } from "react";
import {
  Sparkles,
  Play,
  RotateCcw,
  Film,
  Users,
  Flame,
  Zap,
  Check,
  Copy,
  Download,
  Activity,
  TrendingUp,
  HelpCircle,
  Sun,
  Moon,
} from "lucide-react";
import {
  ResponsiveContainer,
  BarChart,
  Bar,
  XAxis,
  YAxis,
  Tooltip as RechartsTooltip,
  Cell,
  PieChart,
  Pie,
} from "recharts";

const API_BASE = process.env.REACT_APP_API_URL || "http://localhost:8000";
const POLL_INTERVAL_MS = 500;
const POLL_TIMEOUT_MS = 15 * 60 * 1000; // LLM runs on a rate-limited Groq tier can take minutes

const sleep = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms));

async function fetchJSON(url: string, init?: RequestInit) {
  const res = await fetch(url, init);
  const body = await res.json().catch(() => ({}));
  if (!res.ok) {
    const detail = Array.isArray(body.detail)
      ? body.detail.map((d: { msg: string }) => d.msg).join("; ")
      : body.detail;
    throw new Error(detail || `Request failed (${res.status})`);
  }
  return body;
}

type Action = "like" | "share" | "comment" | "skip" | "never";

interface Persona {
  id: number;
  name: string;
  age: number;
  gender: string;
  location: string;
  followers: number;
  interests: string[];
  behavior: string;
  inTarget: boolean;
  shown: boolean;
  action: Action;
  watch: number;
  round: number;
  x: number;
  y: number;
  why?: string;
}

interface Edge {
  from: number;
  to: number;
  type: "direct" | "algo";
  round: number;
}

interface Breakdown {
  likes: number;
  shares: number;
  comments: number;
  scrolls: number;
}

interface Analysis {
  audience_topics: string[];
  audience_age: string;
  audience_regions: string[];
  content_topics: string[];
  audience_fit: number; // -1 when the audience has no detectable interests
  hook_score: number;
  hook_signals: string[];
  recommendations: string[];
  hook_skip_rate: number;
  relevant_skip_rate: number;
  in_target_engagement: number;
  out_target_engagement: number;
  amplification: number;
  engine: "llm" | "heuristic" | "mixed";
  verdict_distribution: Record<string, number>;
  confidence: number; // -1 for LLM runs
  replicates: number;
}

interface Verdict {
  head: string;
  sub: string;
  verdict: string;
  reached: number;
  inPct: number;
  shareRate: number;
  shares: number;
  shownCount: number;
  virality: number;
  algoScore: number;
  engagement: number;
}

const ACTION_CONFIG_LIGHT: Record<
  Action,
  { label: string; badge: string; color: string; fill: string; icon: string }
> = {
  share: {
    label: "SHARED",
    badge: "bg-emerald-50 text-emerald-800 border-emerald-200 font-bold",
    color: "text-emerald-600",
    fill: "#059669",
    icon: "↗",
  },
  like: {
    label: "LIKED",
    badge: "bg-sky-50 text-sky-800 border-sky-200 font-bold",
    color: "text-sky-600",
    fill: "#0284C7",
    icon: "♥",
  },
  comment: {
    label: "COMMENTED",
    badge: "bg-purple-50 text-purple-800 border-purple-200 font-bold",
    color: "text-purple-600",
    fill: "#7C3AED",
    icon: "◉",
  },
  skip: {
    label: "SKIPPED",
    badge: "bg-slate-100 text-slate-600 border-slate-200 font-medium",
    color: "text-slate-500",
    fill: "#94A3B8",
    icon: "✕",
  },
  never: {
    label: "UNREACHED",
    badge: "bg-slate-50 text-slate-400 border-slate-200 font-normal",
    color: "text-slate-400",
    fill: "#E2E8F0",
    icon: "·",
  },
};

const ACTION_CONFIG_DARK: Record<
  Action,
  { label: string; badge: string; color: string; fill: string; icon: string }
> = {
  share: {
    label: "SHARED",
    badge: "bg-emerald-500/20 text-emerald-300 border-emerald-500/40 font-bold",
    color: "text-emerald-400",
    fill: "#10B981",
    icon: "↗",
  },
  like: {
    label: "LIKED",
    badge: "bg-sky-500/20 text-sky-300 border-sky-500/40 font-bold",
    color: "text-sky-400",
    fill: "#38BDF8",
    icon: "♥",
  },
  comment: {
    label: "COMMENTED",
    badge: "bg-purple-500/20 text-purple-300 border-purple-500/40 font-bold",
    color: "text-purple-400",
    fill: "#A855F7",
    icon: "◉",
  },
  skip: {
    label: "SKIPPED",
    badge: "bg-slate-800 text-slate-300 border-slate-700 font-medium",
    color: "text-slate-400",
    fill: "#64748B",
    icon: "✕",
  },
  never: {
    label: "UNREACHED",
    badge: "bg-slate-900 text-slate-500 border-slate-800 font-normal",
    color: "text-slate-500",
    fill: "#334155",
    icon: "·",
  },
};

const CAMPAIGN_PRESETS = [
  {
    tag: "🔥 B2B SaaS",
    title: "AI Growth Playbook",
    tabStyle: "bg-[#F7F3EB] hover:bg-[#EFE7D8] text-[#5C4328] border-[#E5DAC4] dark:bg-amber-950/30 dark:hover:bg-amber-950/50 dark:text-amber-300 dark:border-amber-800/60",
    badgeStyle: "bg-[#EFE7D8] text-[#4A351E] dark:bg-amber-900/50 dark:text-amber-200",
    demographic: "Gen Z tech founders & SaaS product managers looking for viral growth strategies",
    content: "Revealing the exact 3-step distribution playbook we used to scale an AI tool from 0 to 50k MRR without paid ads. Hook: Stop building features nobody asked for.",
  },
  {
    tag: "🤖 Dev Tools",
    title: "Trace Visualizer",
    tabStyle: "bg-[#EEF7F2] hover:bg-[#E2F0E8] text-[#1E4A35] border-[#CFE5D8] dark:bg-emerald-950/30 dark:hover:bg-emerald-950/50 dark:text-emerald-300 dark:border-emerald-800/60",
    badgeStyle: "bg-[#D8EFE2] text-[#163A29] dark:bg-emerald-900/50 dark:text-emerald-200",
    demographic: "Full-stack software engineers, DevOps leads, and Indie Hackers building in public",
    content: "Why 90% of developers waste 4 hours every week debugging microservices. Introducing an automated trace visualizer that spots bottlenecks in 1 click.",
  },
  {
    tag: "📱 DTC Product",
    title: "Eco Coffee Drop",
    tabStyle: "bg-[#FDF2ED] hover:bg-[#FAE4D9] text-[#7A361D] border-[#F5D5C4] dark:bg-orange-950/30 dark:hover:bg-orange-950/50 dark:text-orange-300 dark:border-orange-800/60",
    badgeStyle: "bg-[#F7DACB] text-[#612A15] dark:bg-orange-900/50 dark:text-orange-200",
    demographic: "Eco-conscious millennial urban professionals & sustainable lifestyle creators",
    content: "We tested 100 plastic-free coffee pods so you don't have to. Here is the only one that actually brews espresso crema without degrading the planet.",
  },
  {
    tag: "💡 Creator Hook",
    title: "Short-Form Formula",
    tabStyle: "bg-[#F5F2FC] hover:bg-[#ECE5FA] text-[#4A2F8A] border-[#DFD6F5] dark:bg-purple-950/30 dark:hover:bg-purple-950/50 dark:text-purple-300 dark:border-purple-800/60",
    badgeStyle: "bg-[#E6DBFA] text-[#392170] dark:bg-purple-900/50 dark:text-purple-200",
    demographic: "Solo content creators, YouTube shorts editors, and digital agency operators",
    content: "The psychology behind 3-second visual hooks that stop doomscrolling on TikTok & Instagram Reels. Master the pattern interrupt framework.",
  },
];

// Below this canvas width the two cohorts are stacked vertically instead of side by side.
const NARROW_CANVAS = 640;

function canvasSize(canvas: HTMLCanvasElement) {
  const W = canvas.offsetWidth || 900;
  return { W, H: W < NARROW_CANVAS ? 720 : 540 };
}

interface Region {
  x0: number;
  x1: number;
  y0: number;
  y1: number;
}

// Where each cohort's nodes live (below the cohort header badges).
function cohortRegions(W: number, H: number): { inTarget: Region; outTarget: Region } {
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

function layoutPersonas(ps: Persona[], W: number, H: number) {
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

function setupCanvas(canvas: HTMLCanvasElement) {
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
function drawBackdrop(ctx: CanvasRenderingContext2D, W: number, H: number, isDark: boolean) {
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

function drawRound(
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

export default function App() {
  const [darkMode, setDarkMode] = useState(false);
  const [fileName, setFileName] = useState<string | null>(null);
  const [videoPreview, setVideoPreview] = useState<string | null>(null);
  const [demographic, setDemographic] = useState("");
  const [contentDescription, setContentDescription] = useState("");
  const [population, setPopulation] = useState(100);
  const [boost, setBoost] = useState(6);
  const [running, setRunning] = useState(false);
  const [simDone, setSimDone] = useState(false);
  const [runId, setRunId] = useState("");
  const [currentRound, setCurrentRound] = useState(0);
  const [spreadStatus, setSpreadStatus] = useState("Ready to simulate audience reaction");
  const [selectedPersona, setSelectedPersona] = useState<Persona | null>(null);
  const [hoveredPersona, setHoveredPersona] = useState<Persona | null>(null);
  const [verdict, setVerdict] = useState<Verdict | null>(null);
  const [breakdown, setBreakdown] = useState<Breakdown | null>(null);
  const [copied, setCopied] = useState(false);
  const [analysis, setAnalysis] = useState<Analysis | null>(null);

  const canvasRef = useRef<HTMLCanvasElement>(null);
  const personasRef = useRef<Persona[]>([]);
  const edgesRef = useRef<Edge[]>([]);
  const selectedIdRef = useRef<number | null>(null);
  const hoveredIdRef = useRef<number | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const videoPreviewUrlRef = useRef<string | null>(null);
  const timersRef = useRef<ReturnType<typeof setTimeout>[]>([]);
  // Incremented on every new run / reset so stale polls and animations stop.
  const runTokenRef = useRef(0);

  const clearTimers = () => {
    timersRef.current.forEach(clearTimeout);
    timersRef.current = [];
  };

  const schedule = (fn: () => void, ms: number) => {
    timersRef.current.push(setTimeout(fn, ms));
  };

  useEffect(() => {
    const runToken = runTokenRef;
    const timers = timersRef;
    return () => {
      runToken.current++;
      timers.current.forEach(clearTimeout);
      if (videoPreviewUrlRef.current) {
        URL.revokeObjectURL(videoPreviewUrlRef.current);
      }
    };
  }, []);

  const redraw = useCallback(
    (round: number, selId: number | null, hovId: number | null) => {
      if (canvasRef.current && personasRef.current.length) {
        drawRound(
          canvasRef.current,
          personasRef.current,
          edgesRef.current,
          round,
          selId,
          hovId,
          darkMode
        );
      }
    },
    [darkMode]
  );

  const handleFile = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    setFileName(file.name);
    if (videoPreviewUrlRef.current) {
      URL.revokeObjectURL(videoPreviewUrlRef.current);
    }
    const url = URL.createObjectURL(file);
    videoPreviewUrlRef.current = url;
    setVideoPreview(url);
  };

  const applyPreset = (preset: typeof CAMPAIGN_PRESETS[0]) => {
    setDemographic(preset.demographic);
    setContentDescription(preset.content);
  };

  const runSim = async () => {
    if (running) return;
    clearTimers();
    const token = ++runTokenRef.current;
    const isStale = () => token !== runTokenRef.current;

    setRunning(true);
    setSimDone(false);
    setSelectedPersona(null);
    setVerdict(null);
    setBreakdown(null);
    setAnalysis(null);
    selectedIdRef.current = null;

    const formData = new FormData();
    formData.append("demographic", demographic.trim() || "General audience");
    formData.append("population", String(population));
    formData.append("boost", String(boost));
    if (fileInputRef.current?.files?.[0]) {
      formData.append("video", fileInputRef.current.files[0]);
    }
    // Only send the real hook text — substituting the demographic here would make
    // every post look like a perfect audience match.
    const descriptionText = contentDescription.trim();
    if (descriptionText) {
      formData.append("content_description", descriptionText);
    }

    let jobId: string;
    try {
      const job = await fetchJSON(`${API_BASE}/api/simulate`, {
        method: "POST",
        body: formData,
      });
      jobId = job.job_id;
    } catch (err) {
      if (isStale()) return;
      setRunning(false);
      setSpreadStatus(
        err instanceof TypeError
          ? `Failed to connect to API at ${API_BASE}`
          : `Error: ${(err as Error).message}`
      );
      return;
    }
    if (isStale()) return;

    setRunId(`${jobId.slice(0, 8)} · ${population} agents`);
    setSpreadStatus(`Generating ${population} synthetic personas...`);

    // Poll sequentially (never overlapping requests) until done, failed, or timed out.
    let resultData: any;
    try {
      const deadline = Date.now() + POLL_TIMEOUT_MS;
      for (;;) {
        await sleep(POLL_INTERVAL_MS);
        if (isStale()) return;
        if (Date.now() > deadline) {
          throw new Error("Timed out waiting for the worker — is the Go worker running?");
        }
        const statusData = await fetchJSON(`${API_BASE}/api/status/${jobId}`);
        if (isStale()) return;
        if (statusData.status === "running") {
          setSpreadStatus(`Evaluating ${population} personas...`);
        } else if (statusData.status === "failed") {
          throw new Error("Simulation failed — check worker logs");
        } else if (statusData.status === "complete") {
          resultData = await fetchJSON(`${API_BASE}/api/results/${jobId}`);
          break;
        }
      }
    } catch (err) {
      if (isStale()) return;
      setRunning(false);
      setSpreadStatus(
        err instanceof TypeError
          ? `Lost connection to API at ${API_BASE}`
          : (err as Error).message
      );
      return;
    }
    if (isStale()) return;

    const ps: Persona[] = resultData.personas.map((p: any) => ({
      id: p.persona_id,
      name: p.name,
      age: p.age,
      gender: p.gender || "unknown",
      location: p.location,
      followers: p.followers || 0,
      interests: p.interests || [],
      behavior: p.behavior || "",
      inTarget: p.in_target,
      shown: p.shown !== false,
      action: p.action as Action,
      watch: p.watch,
      round: p.round || (p.shown ? 1 : 0),
      why: p.why,
      x: 0,
      y: 0,
    }));

    const { W, H } = canvasRef.current ? canvasSize(canvasRef.current) : { W: 900, H: 540 };
    layoutPersonas(ps, W, H);
    personasRef.current = ps;

    edgesRef.current = (resultData.edges || []).map((e: any) => ({
      from: e.from - 1,
      to: e.to - 1,
      type: e.type,
      round: e.round,
    }));

    setBreakdown(resultData.breakdown);
    setRunning(false);
    setSpreadStatus("Round 1: Initial Hook & Seed Evaluation");
    setCurrentRound(1);
    redraw(1, null, null);

    schedule(() => {
      setCurrentRound(2);
      setSpreadStatus("Round 2: Direct Share Cascade");
      schedule(() => {
        setCurrentRound(3);
        setSpreadStatus("Round 3: Algorithmic Recommendation Push");
        setSimDone(true);

        setAnalysis(resultData.analysis ?? null);
        setVerdict({
          head: resultData.verdict.head,
          sub: resultData.verdict.sub,
          verdict: resultData.verdict.verdict,
          reached: resultData.personas.filter((x: any) => x.shown).length,
          inPct: resultData.verdict.in_pct,
          shareRate: resultData.verdict.share_rate,
          shares: resultData.breakdown.shares,
          shownCount: resultData.personas.length,
          virality: resultData.virality,
          algoScore: resultData.algo_score,
          engagement: resultData.engagement,
        });
      }, 600);
    }, 600);
  };

  const handleCanvasClick = (e: React.MouseEvent<HTMLCanvasElement>) => {
    if (!canvasRef.current || !personasRef.current.length) return;
    const rect = canvasRef.current.getBoundingClientRect();
    const cx = e.clientX - rect.left;
    const cy = e.clientY - rect.top;
    let hit: Persona | null = null;
    let minD = Infinity;

    personasRef.current.forEach((p) => {
      const d = Math.hypot(p.x - cx, p.y - cy);
      if (d < 22 && d < minD) {
        minD = d;
        hit = p;
      }
    });

    if (hit) {
      selectedIdRef.current = (hit as Persona).id;
      setSelectedPersona(hit);
    } else {
      selectedIdRef.current = null;
      setSelectedPersona(null);
    }
    redraw(currentRound, selectedIdRef.current, hoveredIdRef.current);
  };

  const handleCanvasMouseMove = (e: React.MouseEvent<HTMLCanvasElement>) => {
    if (!canvasRef.current || !personasRef.current.length) return;
    const rect = canvasRef.current.getBoundingClientRect();
    const cx = e.clientX - rect.left;
    const cy = e.clientY - rect.top;
    let hit: Persona | null = null;
    let minD = Infinity;

    personasRef.current.forEach((p) => {
      const d = Math.hypot(p.x - cx, p.y - cy);
      if (d < 20 && d < minD) {
        minD = d;
        hit = p;
      }
    });

    if (hit !== hoveredPersona) {
      hoveredIdRef.current = hit ? (hit as Persona).id : null;
      setHoveredPersona(hit);
      redraw(currentRound, selectedIdRef.current, hoveredIdRef.current);
    }
  };

  const replay = () => {
    clearTimers();
    setCurrentRound(1);
    setSpreadStatus("Round 1: Initial Hook Evaluation");
    schedule(() => {
      setCurrentRound(2);
      setSpreadStatus("Round 2: Direct Share Cascade");
      schedule(() => {
        setCurrentRound(3);
        setSpreadStatus("Round 3: Algorithmic Push");
      }, 900);
    }, 900);
  };

  const scrubRound = (e: React.MouseEvent<HTMLDivElement>) => {
    const rect = (e.currentTarget as HTMLDivElement).getBoundingClientRect();
    const pct = (e.clientX - rect.left) / rect.width;
    const r = Math.max(1, Math.min(3, Math.round(pct * 3)));
    setCurrentRound(r);
    redraw(r, selectedIdRef.current, hoveredIdRef.current);
    setSpreadStatus(`Round ${r} / 3`);
  };

  const drawEmptyCanvas = useCallback(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const setup = setupCanvas(canvas);
    if (!setup) return;
    const { ctx, W, H } = setup;
    drawBackdrop(ctx, W, H, darkMode);

    ctx.fillStyle = darkMode ? "#94A3B8" : "#475569";
    ctx.font = "bold 15px Inter, sans-serif";
    ctx.textAlign = "center";
    const midY = W < NARROW_CANVAS ? H / 4 + 30 : H / 2;
    ctx.fillText("Audience Spread Graph Ready", W / 2, midY - 14);
    ctx.font = "500 12.5px Inter, sans-serif";
    ctx.fillStyle = "#64748B";
    ctx.fillText(
      W < NARROW_CANVAS
        ? "Pick a preset, then Run Simulation"
        : "Select a campaign preset or enter prompt, then click RUN SIMULATION",
      W / 2,
      midY + 14
    );
    ctx.textAlign = "left";
  }, [darkMode]);

  const resetAll = () => {
    clearTimers();
    runTokenRef.current++;
    setRunning(false);
    personasRef.current = [];
    edgesRef.current = [];
    if (videoPreviewUrlRef.current) {
      URL.revokeObjectURL(videoPreviewUrlRef.current);
      videoPreviewUrlRef.current = null;
    }
    setVideoPreview(null);
    setFileName(null);
    if (fileInputRef.current) fileInputRef.current.value = "";
    setRunId("");
    setSimDone(false);
    setVerdict(null);
    setBreakdown(null);
    setAnalysis(null);
    setSelectedPersona(null);
    setCurrentRound(0);
    setSpreadStatus("Ready to simulate audience reaction");
    drawEmptyCanvas();
  };

  useEffect(() => {
    // Re-theme the placeholder only; an existing graph is redrawn by the effect below.
    if (!personasRef.current.length) drawEmptyCanvas();
  }, [drawEmptyCanvas]);

  useEffect(() => {
    if (currentRound > 0)
      redraw(currentRound, selectedIdRef.current, hoveredIdRef.current);
  }, [currentRound, redraw]);

  // Re-layout and redraw when the canvas width changes (window resize, rotation).
  const currentRoundRef = useRef(currentRound);
  currentRoundRef.current = currentRound;
  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas || typeof ResizeObserver === "undefined") return;
    let lastW = canvas.offsetWidth;
    const ro = new ResizeObserver(() => {
      if (canvas.offsetWidth === lastW) return;
      lastW = canvas.offsetWidth;
      if (personasRef.current.length) {
        const { W, H } = canvasSize(canvas);
        layoutPersonas(personasRef.current, W, H);
        redraw(currentRoundRef.current || 3, selectedIdRef.current, hoveredIdRef.current);
      } else {
        drawEmptyCanvas();
      }
    });
    ro.observe(canvas);
    return () => ro.disconnect();
  }, [redraw, drawEmptyCanvas]);

  const copySummary = () => {
    if (!verdict) return;
    const text = `Viralyst Simulation Report [${runId}]
Headline: ${verdict.head} (${verdict.verdict})
Summary: ${verdict.sub}
Reach: ${verdict.reached}/${population} (${verdict.inPct}% In-Target)
Share Rate: ${verdict.shareRate}% (${verdict.shares} organic shares)
Virality Index: ${verdict.virality}/100`;
    navigator.clipboard
      .writeText(text)
      .then(() => {
        setCopied(true);
        setTimeout(() => setCopied(false), 2000);
      })
      .catch(() => setCopied(false));
  };

  const exportJSON = () => {
    if (!personasRef.current.length) return;
    const data = {
      runId,
      verdict,
      analysis,
      breakdown,
      personas: personasRef.current,
    };
    const blob = new Blob([JSON.stringify(data, null, 2)], {
      type: "application/json",
    });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `viralyst-simulation-${Date.now()}.json`;
    a.click();
    URL.revokeObjectURL(url);
  };

  const actionConfig = darkMode ? ACTION_CONFIG_DARK : ACTION_CONFIG_LIGHT;

  const funnelData = useMemo(() => {
    if (!breakdown || !verdict) return [];
    return [
      { name: "Reached", value: verdict.reached, fill: "#4F46E5" },
      { name: "Likes", value: breakdown.likes, fill: "#0284C7" },
      { name: "Comments", value: breakdown.comments, fill: "#7C3AED" },
      { name: "Shares", value: breakdown.shares, fill: "#059669" },
    ];
  }, [breakdown, verdict]);

  const demographicPieData = useMemo(() => {
    if (!verdict) return [];
    return [
      { name: "In-Target", value: verdict.inPct, fill: "#4F46E5" },
      { name: "Out-of-Target", value: 100 - verdict.inPct, fill: "#F43F5E" },
    ];
  }, [verdict]);

  return (
    <div
      className={`min-h-screen transition-colors duration-200 ${
        darkMode
          ? "dark text-slate-100 bg-[#070A12] selection:bg-cyan-500 selection:text-black"
          : "text-slate-900 bg-gradient-to-br from-[#FAF8F5] via-[#F4EFE6]/40 to-[#EFF4FA] selection:bg-indigo-500 selection:text-white"
      }`}
      style={{ fontFamily: "Inter, system-ui, sans-serif" }}
    >
      {/* Top Navbar */}
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

      <main className="max-w-7xl mx-auto px-4 sm:px-6 py-6 space-y-6">
        {/* Preset Quick Fill Tabs (Warm Beige, Sage, Terracotta, Lavender) */}
        <div className="flex items-center gap-2 overflow-x-auto pb-1 text-xs">
          <span
            className={`font-bold uppercase tracking-wider text-[10px] flex items-center gap-1 shrink-0 ${
              darkMode ? "text-slate-400" : "text-[#756550]"
            }`}
          >
            <Sparkles className="w-3.5 h-3.5 text-indigo-500" /> Quick Presets:
          </span>
          {CAMPAIGN_PRESETS.map((preset) => (
            <button
              key={preset.title}
              onClick={() => applyPreset(preset)}
              className={`shrink-0 border px-3 py-1.5 rounded-xl text-xs shadow-xs hover:shadow-sm transition flex items-center gap-1.5 font-medium ${preset.tabStyle}`}
            >
              <span className={`px-1.5 py-0.5 rounded-md font-bold text-[10px] ${preset.badgeStyle}`}>
                {preset.tag}
              </span>
              <span>{preset.title}</span>
            </button>
          ))}
        </div>

        {/* SECTION 01: CAMPAIGN CONFIG */}
        <section
          className={`border rounded-2xl shadow-xs overflow-hidden ${
            darkMode
              ? "border-slate-800/90 bg-slate-900/60 backdrop-blur-xl"
              : "border-[#E5DEC9] bg-white/95 backdrop-blur-md"
          }`}
        >
          <div
            className={`px-5 py-3 border-b flex items-center justify-between ${
              darkMode
                ? "border-slate-800 bg-slate-950/60"
                : "border-[#EAE3D2] bg-gradient-to-r from-[#FBF9F4] via-[#F4EFE6]/60 to-white"
            }`}
          >
            <div className="flex items-center gap-2 text-xs font-bold tracking-wider">
              <span className="w-5 h-5 rounded-lg bg-indigo-600 text-white flex items-center justify-center text-[10px] font-black shadow-xs">
                01
              </span>
              <span className={darkMode ? "text-slate-200" : "text-[#3D3021]"}>
                CAMPAIGN HOOK & AUDIENCE SETUP
              </span>
            </div>
            <span className="hidden sm:inline text-[11px] text-slate-500 font-semibold">
              Describe the audience and the first 3 seconds
            </span>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 xl:[grid-template-columns:1.1fr_1.2fr_1.4fr_auto_auto]">
            {/* Video / Creative Uploader */}
            <div
              className={`p-4 border-b xl:border-b-0 md:border-r flex flex-col justify-between ${
                darkMode ? "border-slate-800" : "border-[#EAE3D2]"
              }`}
            >
              <div>
                <div
                  className={`text-[11px] font-bold tracking-wider mb-2 flex items-center gap-1.5 ${
                    darkMode ? "text-slate-400" : "text-[#695B48]"
                  }`}
                >
                  <Film className="w-3.5 h-3.5 text-indigo-500" /> CONTENT VIDEO
                </div>
                <div
                  onClick={() => !videoPreview && fileInputRef.current?.click()}
                  className={`border-2 border-dashed rounded-xl p-3 text-center cursor-pointer transition ${
                    darkMode
                      ? "border-slate-700 bg-slate-950/50 hover:bg-slate-950/80 hover:border-indigo-400"
                      : "border-[#DCD3BE] bg-[#FAF8F5]/80 hover:bg-[#F3EFE6] hover:border-indigo-400"
                  }`}
                >
                  {videoPreview ? (
                    <div className="space-y-2">
                      <div className="overflow-hidden rounded-lg border border-slate-700 h-24 bg-black">
                        <video
                          key={videoPreview}
                          src={videoPreview}
                          controls
                          className="w-full h-full object-cover"
                        />
                      </div>
                      <div className="flex items-center justify-between text-left">
                        <span className="text-xs font-bold truncate max-w-[120px]">
                          {fileName}
                        </span>
                        <span className="text-[10px] text-emerald-800 bg-emerald-100 border border-emerald-300 px-1.5 py-0.5 rounded font-mono font-bold">
                          Ready
                        </span>
                      </div>
                      <button
                        onClick={(e) => {
                          e.stopPropagation();
                          if (fileInputRef.current) fileInputRef.current.value = "";
                          fileInputRef.current?.click();
                        }}
                        className={`text-[11px] w-full border rounded-lg py-1 transition font-semibold ${
                          darkMode
                            ? "border-slate-700 hover:bg-slate-800 text-slate-200"
                            : "border-[#D6CCA8] hover:bg-[#EBE3CF] text-[#4A3924]"
                        }`}
                      >
                        Replace Video
                      </button>
                    </div>
                  ) : (
                    <div className="flex flex-col items-center justify-center py-5">
                      <Film className="w-6 h-6 text-slate-400 mb-1.5" />
                      <div
                        className={`text-xs font-bold ${
                          darkMode ? "text-slate-200" : "text-[#4A3C2A]"
                        }`}
                      >
                        Upload Video Hook
                      </div>
                      <div className="text-[10px] text-slate-500 mt-0.5 font-medium">
                        MP4, MOV, WEBM (Max 50MB)
                      </div>
                    </div>
                  )}
                </div>
                <input
                  ref={fileInputRef}
                  type="file"
                  accept="video/*"
                  className="hidden"
                  onChange={handleFile}
                />
              </div>
            </div>

            {/* Target Demographic */}
            <div
              className={`p-4 border-b xl:border-b-0 md:border-r flex flex-col justify-between ${
                darkMode ? "border-slate-800" : "border-[#EAE3D2]"
              }`}
            >
              <div>
                <div
                  className={`text-[11px] font-bold tracking-wider mb-2 flex items-center gap-1.5 ${
                    darkMode ? "text-slate-400" : "text-[#695B48]"
                  }`}
                >
                  <Users className="w-3.5 h-3.5 text-indigo-500" /> TARGET
                  DEMOGRAPHIC
                </div>
                <textarea
                  value={demographic}
                  onChange={(e) => setDemographic(e.target.value)}
                  placeholder="e.g. Gen Z tech founders, AI developers, and product managers in North America & India"
                  className={`w-full border rounded-xl p-2.5 outline-none resize-none text-xs min-h-[105px] transition ${
                    darkMode
                      ? "border-slate-800 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-900/30 bg-slate-950/70 text-slate-100 placeholder-slate-500"
                      : "border-[#DFD6C4] focus:border-indigo-500 focus:ring-2 focus:ring-indigo-100 bg-[#FCFBF8] text-slate-800 placeholder-slate-400"
                  }`}
                  rows={4}
                />
              </div>
            </div>

            {/* Content Description */}
            <div
              className={`p-4 border-b xl:border-b-0 md:border-r flex flex-col justify-between ${
                darkMode ? "border-slate-800" : "border-[#EAE3D2]"
              }`}
            >
              <div>
                <div
                  className={`text-[11px] font-bold tracking-wider mb-2 flex items-center gap-1.5 ${
                    darkMode ? "text-slate-400" : "text-[#695B48]"
                  }`}
                >
                  <Sparkles className="w-3.5 h-3.5 text-purple-500" /> HOOK &
                  CORE MESSAGE
                </div>
                <textarea
                  value={contentDescription}
                  onChange={(e) => setContentDescription(e.target.value)}
                  placeholder="Describe your first 3 seconds, key insight, and CTA (evaluated by each persona)"
                  className={`w-full border rounded-xl p-2.5 outline-none resize-none text-xs min-h-[105px] transition ${
                    darkMode
                      ? "border-slate-800 focus:border-purple-500 focus:ring-2 focus:ring-purple-900/30 bg-slate-950/70 text-slate-100 placeholder-slate-500"
                      : "border-[#DFD6C4] focus:border-purple-500 focus:ring-2 focus:ring-purple-100 bg-[#FCFBF8] text-slate-800 placeholder-slate-400"
                  }`}
                  rows={4}
                />
              </div>
            </div>

            {/* Population & Boost */}
            <div
              className={`p-4 border-b md:border-b-0 md:border-r xl:min-w-48 flex flex-col justify-between gap-3 ${
                darkMode ? "border-slate-800" : "border-[#EAE3D2]"
              }`}
            >
              <div>
                <div
                  className={`text-[11px] font-bold tracking-wider mb-1.5 flex items-center justify-between ${
                    darkMode ? "text-slate-400" : "text-[#695B48]"
                  }`}
                >
                  <span>AGENT POPULATION</span>

                </div>
                <select
                  value={population}
                  onChange={(e) => setPopulation(+e.target.value)}
                  className={`w-full border rounded-xl px-2.5 py-1.5 text-xs outline-none font-bold focus:border-indigo-500 shadow-xs ${
                    darkMode
                      ? "border-slate-700 bg-slate-950 text-slate-100"
                      : "border-[#DFD6C4] bg-[#FCFBF8] text-slate-800"
                  }`}
                >
                  <option value={50}>50 AI Personas</option>
                  <option value={100}>100 AI Personas</option>
                  <option value={150}>150 AI Personas</option>
                  <option value={200}>200 AI Personas</option>
                  <option value={250}>250 AI Personas</option>
                </select>
              </div>

              <div>
                <div
                  className={`flex items-center justify-between text-[11px] mb-1 font-semibold ${
                    darkMode ? "text-slate-400" : "text-[#695B48]"
                  }`}
                >
                  <span className="flex items-center gap-1">
                    SEED BOOST
                    <span
                      title="Number of initial seed personas tested in Round 1 (Boost × 4)"
                      className="cursor-help text-slate-400"
                    >
                      <HelpCircle className="w-3 h-3" />
                    </span>
                  </span>
                  <span className="font-bold text-indigo-600 dark:text-indigo-400 font-mono">
                    {boost * 4} seeds
                  </span>
                </div>
                <input
                  type="range"
                  min={1}
                  max={10}
                  step={1}
                  value={boost}
                  onChange={(e) => setBoost(+e.target.value)}
                  className="w-full h-1.5 accent-indigo-600 bg-slate-200 dark:bg-slate-700 rounded-lg cursor-pointer"
                />
              </div>
            </div>

            {/* Run Button with Dynamic Gradient */}
            <button
              onClick={runSim}
              disabled={running}
              className="px-8 py-5 xl:py-0 text-xs font-black tracking-wider transition min-w-36 flex flex-col items-center justify-center gap-1.5 text-white relative overflow-hidden group shadow-md shadow-indigo-500/20"
              style={{
                background: running
                  ? "#64748B"
                  : "linear-gradient(135deg, #2563EB 0%, #4F46E5 50%, #7C3AED 100%)",
              }}
            >
              {running ? (
                <>
                  <div className="w-5 h-5 border-2 border-white border-t-transparent rounded-full animate-spin" />
                  <span className="text-[10px] font-mono text-indigo-100">
                    SIMULATING...
                  </span>
                </>
              ) : (
                <>
                  <Zap className="w-5 h-5 text-white group-hover:scale-110 transition-transform" />
                  <span className="text-xs">RUN SIMULATION</span>
                </>
              )}
            </button>
          </div>
        </section>

        {/* SECTION 02: CANVAS SPREAD VISUALIZER */}
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
                    onClick={() => {
                      setSelectedPersona(null);
                      selectedIdRef.current = null;
                      redraw(currentRound, null, hoveredIdRef.current);
                    }}
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

        {/* SECTION 03: VERDICT & METRICS */}
        <section
          className={`border rounded-2xl p-4 sm:p-6 shadow-xs space-y-6 ${
            darkMode
              ? "border-slate-800/90 bg-slate-900/60 backdrop-blur-xl"
              : "border-[#E5DEC9] bg-white/95 backdrop-blur-md"
          }`}
        >
          <div
            className={`flex items-center justify-between border-b pb-3 ${
              darkMode ? "border-slate-800" : "border-[#EAE3D2]"
            }`}
          >
            <div className="flex items-center gap-2 text-xs font-bold tracking-wider">
              <span className="w-5 h-5 rounded-lg bg-purple-600 text-white flex items-center justify-center text-[10px] font-black shadow-xs">
                03
              </span>
              <span className={darkMode ? "text-slate-200" : "text-[#3D3021]"}>
                EXECUTIVE VERDICT & PREDICTIVE KPIS
              </span>
            </div>
            <span className="text-[11px] text-slate-500 font-semibold">
              Algorithm Score & Conversion Funnel
            </span>
          </div>

          {!verdict ? (
            <div className="py-8 text-center">
              <TrendingUp className="w-8 h-8 text-slate-400 mx-auto mb-2" />
              <div
                className={`text-sm font-bold mb-1 ${
                  darkMode ? "text-slate-300" : "text-[#4A3C2A]"
                }`}
              >
                Verdict & Analytics Output Appears Here
              </div>
              <p className="text-xs text-slate-500 font-medium">
                Run a simulation to generate virality index, reach breakdown, and
                actionable hook recommendations.
              </p>
            </div>
          ) : (
            <div className="space-y-6">
              {/* Verdict Header Banner with Rich Gradient */}
              <div
                className={`flex flex-col sm:flex-row sm:items-center justify-between gap-3 p-5 rounded-2xl border ${
                  darkMode
                    ? "bg-gradient-to-r from-slate-950 via-indigo-950/30 to-slate-950 border-slate-800"
                    : "bg-gradient-to-r from-[#FAF7F2] via-[#F4EFE6]/80 to-[#FAF7F2] border-[#DFD6C4]"
                }`}
              >
                <div>
                  <div className="flex items-center gap-3">
                    <h3 className="text-2xl font-black tracking-tight">
                      {verdict.head}
                    </h3>
                    <span
                      className={`text-xs font-extrabold px-3 py-1 rounded-full border shadow-xs ${
                        darkMode
                          ? "bg-indigo-500/20 text-indigo-300 border-indigo-500/40"
                          : "bg-[#EFE9DC] text-[#4A3924] border-[#DFD6C4]"
                      }`}
                    >
                      {verdict.verdict}
                    </span>
                  </div>
                  <p
                    className={`text-xs mt-1.5 leading-relaxed max-w-3xl font-medium ${
                      darkMode ? "text-slate-400" : "text-[#5C4D3B]"
                    }`}
                  >
                    {verdict.sub}
                  </p>
                </div>

                <div className="sm:text-right sm:pl-4">
                  <div className="text-[10px] uppercase font-black text-slate-400 mb-0.5 tracking-wider">
                    Virality Index
                  </div>
                  <div className="text-3xl font-black text-indigo-600 dark:text-indigo-400 font-mono">
                    {verdict.virality}
                    <span className="text-sm text-slate-400 font-normal">/100</span>
                  </div>
                </div>
              </div>

              {/* KPI Cards Grid */}
              <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 gap-3">
                <div
                  className={`p-3.5 rounded-xl border shadow-xs ${
                    darkMode
                      ? "bg-slate-950/60 border-slate-800"
                      : "bg-[#FAF8F5] border-[#E5DEC9]"
                  }`}
                >
                  <div className="text-[10px] font-bold text-slate-500 mb-1">
                    TOTAL REACH
                  </div>
                  <div className="text-xl font-black text-indigo-600 dark:text-indigo-400 font-mono">
                    {verdict.reached}
                  </div>
                  <div className="text-[10px] text-slate-500 mt-1 font-medium">
                    {verdict.reached} of {population} agents
                  </div>
                </div>

                <div
                  className={`p-3.5 rounded-xl border shadow-xs ${
                    darkMode
                      ? "bg-slate-950/60 border-slate-800"
                      : "bg-[#FAF8F5] border-[#E5DEC9]"
                  }`}
                >
                  <div className="text-[10px] font-bold text-slate-500 mb-1">
                    IN / OUT DEMO RATIO
                  </div>
                  <div className="text-xl font-black font-mono">
                    {verdict.inPct}% / {100 - verdict.inPct}%
                  </div>
                  <div
                    className={`h-1.5 rounded-full mt-2 overflow-hidden ${
                      darkMode ? "bg-slate-800" : "bg-slate-200"
                    }`}
                  >
                    <div
                      className="h-full bg-indigo-600 rounded-full"
                      style={{ width: `${verdict.inPct}%` }}
                    />
                  </div>
                </div>

                <div
                  className={`p-3.5 rounded-xl border shadow-xs ${
                    darkMode
                      ? "bg-slate-950/60 border-slate-800"
                      : "bg-[#FAF8F5] border-[#E5DEC9]"
                  }`}
                >
                  <div className="text-[10px] font-bold text-slate-500 mb-1">
                    ORGANIC SHARE RATE
                  </div>
                  <div className="text-xl font-black text-emerald-600 dark:text-emerald-400 font-mono">
                    {verdict.shareRate}%
                  </div>
                  <div className="text-[10px] text-slate-500 mt-1 font-medium">
                    {verdict.shares} total shares
                  </div>
                </div>

                <div
                  className={`p-3.5 rounded-xl border shadow-xs ${
                    darkMode
                      ? "bg-slate-950/60 border-slate-800"
                      : "bg-[#FAF8F5] border-[#E5DEC9]"
                  }`}
                >
                  <div className="text-[10px] font-bold text-slate-500 mb-1">
                    ALGORITHM SCORE
                  </div>
                  <div className="text-xl font-black text-purple-600 dark:text-purple-400 font-mono">
                    {verdict.algoScore}
                  </div>
                  <div className="text-[10px] text-slate-500 mt-1 font-medium">
                    Push Probability
                  </div>
                </div>

                <div
                  className={`p-3.5 rounded-xl border shadow-xs ${
                    darkMode
                      ? "bg-slate-950/60 border-slate-800"
                      : "bg-[#FAF8F5] border-[#E5DEC9]"
                  }`}
                >
                  <div className="text-[10px] font-bold text-slate-500 mb-1">
                    ENGAGEMENT %
                  </div>
                  <div className="text-xl font-black text-amber-600 dark:text-amber-400 font-mono">
                    {verdict.engagement}%
                  </div>
                  <div className="text-[10px] text-slate-500 mt-1 font-medium">
                    Non-skip rate
                  </div>
                </div>
              </div>

              {analysis && <AnalysisPanel analysis={analysis} darkMode={darkMode} />}

              {/* Recharts Analytics Charts */}
              <div className="grid grid-cols-1 lg:grid-cols-2 gap-4 pt-2">
                <div
                  className={`p-4 rounded-xl border shadow-xs ${
                    darkMode
                      ? "bg-slate-950/60 border-slate-800"
                      : "bg-[#FAF8F5] border-[#E5DEC9]"
                  }`}
                >
                  <div className="text-xs font-bold mb-3 flex items-center justify-between">
                    <span>AUDIENCE ENGAGEMENT FUNNEL</span>
                    <span className="text-[10px] text-slate-500 font-mono font-medium">
                      Action Breakdown
                    </span>
                  </div>
                  <div className="h-44">
                    <ResponsiveContainer width="100%" height="100%">
                      <BarChart data={funnelData} layout="vertical">
                        <XAxis
                          type="number"
                          stroke={darkMode ? "#64748B" : "#94A3B8"}
                          fontSize={10}
                        />
                        <YAxis
                          type="category"
                          dataKey="name"
                          stroke={darkMode ? "#94A3B8" : "#64748B"}
                          fontSize={11}
                          width={70}
                        />
                        <RechartsTooltip
                          contentStyle={{
                            backgroundColor: darkMode ? "#0F172A" : "#FFFFFF",
                            borderColor: darkMode ? "#334155" : "#E2E8F0",
                            color: darkMode ? "#F8FAFC" : "#0F172A",
                            borderRadius: "10px",
                            fontSize: "11px",
                            boxShadow: "0 10px 15px -3px rgb(0 0 0 / 0.1)",
                          }}
                        />
                        <Bar dataKey="value" radius={[0, 6, 6, 0]}>
                          {funnelData.map((entry, index) => (
                            <Cell key={`cell-${index}`} fill={entry.fill} />
                          ))}
                        </Bar>
                      </BarChart>
                    </ResponsiveContainer>
                  </div>
                </div>

                <div
                  className={`p-4 rounded-xl border shadow-xs ${
                    darkMode
                      ? "bg-slate-950/60 border-slate-800"
                      : "bg-[#FAF8F5] border-[#E5DEC9]"
                  }`}
                >
                  <div className="text-xs font-bold mb-3 flex items-center justify-between">
                    <span>COHORT SPLIT (IN-TARGET VS OUT-OF-TARGET)</span>
                    <span className="text-[10px] text-slate-500 font-mono font-medium">
                      Spread Distribution
                    </span>
                  </div>
                  <div className="h-44 flex items-center justify-center">
                    <div className="h-full flex-1 min-w-0">
                    <ResponsiveContainer width="100%" height="100%">
                      <PieChart>
                        <Pie
                          data={demographicPieData}
                          innerRadius={45}
                          outerRadius={65}
                          paddingAngle={5}
                          dataKey="value"
                        >
                          {demographicPieData.map((entry, index) => (
                            <Cell key={`pie-cell-${index}`} fill={entry.fill} />
                          ))}
                        </Pie>
                        <RechartsTooltip
                          contentStyle={{
                            backgroundColor: darkMode ? "#0F172A" : "#FFFFFF",
                            borderColor: darkMode ? "#334155" : "#E2E8F0",
                            color: darkMode ? "#F8FAFC" : "#0F172A",
                            borderRadius: "10px",
                            fontSize: "11px",
                            boxShadow: "0 10px 15px -3px rgb(0 0 0 / 0.1)",
                          }}
                        />
                      </PieChart>
                    </ResponsiveContainer>
                    </div>
                    <div className="space-y-2 pr-2 sm:pr-6 shrink-0">
                      <div className="flex items-center gap-2 text-xs">
                        <span className="w-2.5 h-2.5 rounded-full bg-[#4F46E5]" />
                        <span className="font-bold">
                          In-Target: {verdict.inPct}%
                        </span>
                      </div>
                      <div className="flex items-center gap-2 text-xs">
                        <span className="w-2.5 h-2.5 rounded-full bg-[#F43F5E]" />
                        <span className="font-bold">
                          Out-of-Target: {100 - verdict.inPct}%
                        </span>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          )}
        </section>
      </main>
    </div>
  );
}

function Chips({ items, empty, tone }: { items: string[]; empty: string; tone: string }) {
  if (!items.length) {
    return <span className="text-[11px] italic text-slate-500">{empty}</span>;
  }
  return (
    <div className="flex flex-wrap gap-1">
      {items.map((t) => (
        <span key={t} className={`text-[10px] border px-2 py-0.5 rounded-md font-semibold ${tone}`}>
          {t}
        </span>
      ))}
    </div>
  );
}

function ScoreBar({ label, value, hint }: { label: string; value: number; hint?: string }) {
  const color = value >= 65 ? "bg-emerald-500" : value >= 35 ? "bg-amber-500" : "bg-rose-500";
  return (
    <div>
      <div className="flex justify-between text-[11px] font-semibold mb-1">
        <span className="text-slate-500">{label}</span>
        <span className="font-mono font-bold">{value < 0 ? "n/a" : `${value}/100`}</span>
      </div>
      <div className="h-1.5 rounded-full overflow-hidden bg-slate-200 dark:bg-slate-800">
        {value >= 0 && <div className={`h-full rounded-full ${color}`} style={{ width: `${value}%` }} />}
      </div>
      {hint && <div className="text-[10px] text-slate-500 mt-1">{hint}</div>}
    </div>
  );
}

// Explains which signals from the inputs drove the verdict.
function AnalysisPanel({ analysis: a, darkMode }: { analysis: Analysis; darkMode: boolean }) {
  const card = `p-4 rounded-xl border shadow-xs space-y-3 ${
    darkMode ? "bg-slate-950/60 border-slate-800" : "bg-[#FAF8F5] border-[#E5DEC9]"
  }`;
  const chipTone = darkMode
    ? "bg-slate-900 border-slate-700 text-indigo-300"
    : "bg-white border-[#E8DFC9] text-[#5C4328]";
  const label = "text-[10px] font-bold uppercase tracking-wider text-slate-500 mb-1";
  const verdicts = Object.entries(a.verdict_distribution || {}).sort((x, y) => y[1] - x[1]);

  return (
    <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
      <div className={card}>
        <div className="text-xs font-bold">WHAT THE MODEL READ FROM YOUR INPUTS</div>
        <div>
          <div className={label}>Audience interests</div>
          <Chips items={a.audience_topics} empty="None detected — describe what they care about" tone={chipTone} />
          {(a.audience_age || a.audience_regions.length > 0) && (
            <div className="text-[10px] text-slate-500 mt-1">
              {[a.audience_age, a.audience_regions.join(", ")].filter(Boolean).join(" · ")}
            </div>
          )}
        </div>
        <div>
          <div className={label}>Hook topics</div>
          <Chips items={a.content_topics} empty="None detected" tone={chipTone} />
        </div>
        <ScoreBar
          label="Audience fit"
          value={a.audience_fit}
          hint={a.audience_fit < 0 ? "Audience too vague to measure fit" : "How much the hook speaks to the audience's interests"}
        />
        <ScoreBar label="Hook strength" value={a.hook_score} hint={a.hook_signals.join(" · ") || "No strong hook signals found"} />
      </div>

      <div className={card}>
        <div className="flex items-center justify-between">
          <div className="text-xs font-bold">WHY THIS VERDICT</div>
          <span className="text-[10px] font-mono text-slate-500">
            {a.engine === "llm" ? "LLM personas" : a.engine === "mixed" ? "LLM + offline model" : "Offline model"}
          </span>
        </div>
        <div className="grid grid-cols-3 gap-2 text-center">
          {[
            ["Relevant viewers who skipped", a.relevant_skip_rate >= 0 ? a.relevant_skip_rate : a.hook_skip_rate],
            ["In-target engaged", a.in_target_engagement],
            ["Out-of-target engaged", a.out_target_engagement],
          ].map(([k, v]) => (
            <div key={k as string} className={`rounded-lg border p-2 ${darkMode ? "border-slate-800" : "border-[#E8DFC9] bg-white"}`}>
              <div className="text-base font-black font-mono">{v}%</div>
              <div className="text-[10px] text-slate-500 leading-tight">{k}</div>
            </div>
          ))}
        </div>
        {a.confidence >= 0 && verdicts.length > 0 && (
          <div className="text-[11px]">
            <span className={`font-bold ${a.confidence < 50 ? "text-amber-600 dark:text-amber-400" : ""}`}>
              Confidence {a.confidence}%{a.confidence < 50 ? " · borderline result" : ""}
            </span>
            <span className="text-slate-500">
              {" "}— {verdicts.map(([v, n]) => `${v.replace(/\.$/, "")} ${n}/${a.replicates}`).join(", ")} simulated runs
            </span>
          </div>
        )}
        <div>
          <div className={label}>How to improve</div>
          {a.recommendations.length ? (
            <ul className="space-y-1 text-[11px] list-disc pl-4">
              {a.recommendations.map((r) => (
                <li key={r}>{r}</li>
              ))}
            </ul>
          ) : (
            <span className="text-[11px] text-slate-500">No major issues found in the hook.</span>
          )}
        </div>
      </div>
    </div>
  );
}
