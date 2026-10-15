import React, { useState, useRef, useEffect, useCallback, useMemo, Suspense, lazy } from "react";
import {
  API_BASE,
  POLL_INTERVAL_MS,
  POLL_TIMEOUT_MS,
  sleep,
  fetchJSON,
  parsePersonas,
  parseEdges,
  parseVerdict,
} from "./api";
import { Example } from "./constants";
import { Analysis, Breakdown, Edge, Persona, Verdict } from "./types";
import {
  NARROW_CANVAS,
  canvasSize,
  layoutPersonas,
  setupCanvas,
  drawBackdrop,
  drawRound,
} from "./lib/canvas";
import { buildSummaryText, downloadJSON } from "./lib/report";
import Header from "./components/Header";
import Hero from "./components/Hero";
import SetupPanel from "./components/SetupPanel";
import SpreadGraph, { GraphView } from "./components/SpreadGraph";
import ResultsPanel from "./components/ResultsPanel";

// three.js loads after first paint, in its own chunk.
const Backdrop3D = lazy(() => import("./components/Backdrop3D"));

const THEME_KEY = "viralyst-theme";

// Remembered choice first, then the system setting. Storage can throw (private mode).
function initialDarkMode() {
  try {
    const saved = localStorage.getItem(THEME_KEY);
    if (saved) return saved === "dark";
  } catch {}
  return typeof window.matchMedia === "function" && window.matchMedia("(prefers-color-scheme: dark)").matches;
}

export type RunState = "idle" | "running" | "error" | "done";

export default function App() {
  const [darkMode, setDarkMode] = useState(initialDarkMode);
  const [fileName, setFileName] = useState<string | null>(null);
  const [videoPreview, setVideoPreview] = useState<string | null>(null);
  const [niche, setNiche] = useState("");
  const [demographic, setDemographic] = useState("");
  const [contentDescription, setContentDescription] = useState("");
  const [population, setPopulation] = useState(100);
  const [boost, setBoost] = useState(6);
  const [runState, setRunState] = useState<RunState>("idle");
  const [statusText, setStatusText] = useState("");
  const [runId, setRunId] = useState("");
  const [currentRound, setCurrentRound] = useState(0);
  const [selectedPersona, setSelectedPersona] = useState<Persona | null>(null);
  const [hoveredPersona, setHoveredPersona] = useState<Persona | null>(null);
  const [verdict, setVerdict] = useState<Verdict | null>(null);
  const [breakdown, setBreakdown] = useState<Breakdown | null>(null);
  const [copied, setCopied] = useState(false);
  const [analysis, setAnalysis] = useState<Analysis | null>(null);
  const [view, setView] = useState<GraphView>("globe");
  // The 2D canvas reads personas/edges from refs; the globe needs them as state.
  const [graph, setGraph] = useState<{ personas: Persona[]; edges: Edge[] }>({ personas: [], edges: [] });

  const running = runState === "running";
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

  useEffect(() => {
    document.documentElement.classList.toggle("dark", darkMode);
    try {
      localStorage.setItem(THEME_KEY, darkMode ? "dark" : "light");
    } catch {}
  }, [darkMode]);

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
        drawRound(canvasRef.current, personasRef.current, edgesRef.current, round, selId, hovId, darkMode);
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

  const removeVideo = () => {
    if (videoPreviewUrlRef.current) {
      URL.revokeObjectURL(videoPreviewUrlRef.current);
      videoPreviewUrlRef.current = null;
    }
    setVideoPreview(null);
    setFileName(null);
    if (fileInputRef.current) fileInputRef.current.value = "";
  };

  const applyExample = (example: Example) => {
    setNiche(example.niche);
    setDemographic(example.demographic);
    setContentDescription(example.content);
  };

  const playRounds = (stepMs: number, onDone?: () => void) => {
    clearTimers();
    setCurrentRound(1);
    schedule(() => {
      setCurrentRound(2);
      schedule(() => {
        setCurrentRound(3);
        onDone?.();
      }, stepMs);
    }, stepMs);
  };

  const runSim = async () => {
    if (running) return;
    clearTimers();
    const token = ++runTokenRef.current;
    const isStale = () => token !== runTokenRef.current;

    setRunState("running");
    setSelectedPersona(null);
    setVerdict(null);
    setBreakdown(null);
    setAnalysis(null);
    selectedIdRef.current = null;

    const formData = new FormData();
    formData.append("niche", niche.trim());
    formData.append("demographic", demographic.trim());
    formData.append("population", String(population));
    formData.append("boost", String(boost));
    if (fileInputRef.current?.files?.[0]) {
      formData.append("video", fileInputRef.current.files[0]);
    }
    // Only send the real hook text — substituting the audience here would make
    // every post look like a perfect audience match.
    const descriptionText = contentDescription.trim();
    if (descriptionText) {
      formData.append("content_description", descriptionText);
    }

    const fail = (err: unknown, connectionMsg: string) => {
      if (isStale()) return;
      setRunState("error");
      setStatusText(err instanceof TypeError ? connectionMsg : (err as Error).message);
    };

    setStatusText(fileInputRef.current?.files?.[0] ? "Uploading your video…" : "Queuing simulation…");
    let jobId: string;
    try {
      const job = await fetchJSON(`${API_BASE}/api/simulate`, { method: "POST", body: formData });
      jobId = job.job_id;
    } catch (err) {
      fail(err, `Couldn't reach the API at ${API_BASE}. Is it running?`);
      return;
    }
    if (isStale()) return;

    setRunId(jobId.slice(0, 8));
    setStatusText("Waiting for a worker…");

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
          setStatusText(`${population} personas are watching your post…`);
        } else if (statusData.status === "failed") {
          throw new Error("The simulation failed — check the worker logs.");
        } else if (statusData.status === "complete") {
          resultData = await fetchJSON(`${API_BASE}/api/results/${jobId}`);
          break;
        }
      }
    } catch (err) {
      fail(err, `Lost connection to the API at ${API_BASE}.`);
      return;
    }
    if (isStale()) return;

    const ps = parsePersonas(resultData);
    const { W, H } = canvasRef.current ? canvasSize(canvasRef.current) : { W: 900, H: 460 };
    layoutPersonas(ps, W, H);
    personasRef.current = ps;
    edgesRef.current = parseEdges(resultData);
    setGraph({ personas: ps, edges: edgesRef.current });

    setBreakdown(resultData.breakdown);
    setAnalysis(resultData.analysis ?? null);
    setVerdict(parseVerdict(resultData));
    setRunState("done");
    setStatusText("");
    playRounds(600);
  };

  const hitTest = (e: React.MouseEvent<HTMLCanvasElement>, radius: number): Persona | null => {
    if (!canvasRef.current) return null;
    const rect = canvasRef.current.getBoundingClientRect();
    const cx = e.clientX - rect.left;
    const cy = e.clientY - rect.top;
    let hit: Persona | null = null;
    let minD = Infinity;
    personasRef.current.forEach((p) => {
      const d = Math.hypot(p.x - cx, p.y - cy);
      if (d < radius && d < minD) {
        minD = d;
        hit = p;
      }
    });
    return hit;
  };

  const handleCanvasClick = (e: React.MouseEvent<HTMLCanvasElement>) => {
    if (!personasRef.current.length) return;
    const hit = hitTest(e, 18);
    selectedIdRef.current = hit ? hit.id : null;
    setSelectedPersona(hit);
    redraw(currentRound, selectedIdRef.current, hoveredIdRef.current);
  };

  const handleCanvasMouseMove = (e: React.MouseEvent<HTMLCanvasElement>) => {
    if (!personasRef.current.length) return;
    const hit = hitTest(e, 16);
    if (hit !== hoveredPersona) {
      hoveredIdRef.current = hit ? hit.id : null;
      setHoveredPersona(hit);
      redraw(currentRound, selectedIdRef.current, hoveredIdRef.current);
    }
  };

  const handleCanvasLeave = () => {
    if (!hoveredPersona) return;
    hoveredIdRef.current = null;
    setHoveredPersona(null);
    redraw(currentRound, selectedIdRef.current, null);
  };

  const selectPersona = (p: Persona | null) => {
    selectedIdRef.current = p ? p.id : null;
    setSelectedPersona(p);
    redraw(currentRound, selectedIdRef.current, hoveredIdRef.current);
  };

  const closePersona = () => selectPersona(null);

  const replay = () => playRounds(900);

  const showRound = (r: number) => {
    clearTimers();
    setCurrentRound(r);
  };

  const drawEmptyCanvas = useCallback(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const setup = setupCanvas(canvas);
    if (!setup) return;
    const { ctx, W, H } = setup;
    drawBackdrop(ctx, W, H, darkMode);
    ctx.fillStyle = darkMode ? "rgba(247,247,238,0.45)" : "rgba(41,41,25,0.5)";
    ctx.font = "400 13px Inter, system-ui, sans-serif";
    ctx.textAlign = "center";
    const midY = W < NARROW_CANVAS ? H / 4 + 20 : H / 2;
    ctx.fillText("Each dot will be one simulated viewer", W / 2, midY);
    ctx.textAlign = "left";
  }, [darkMode]);

  const resetAll = () => {
    clearTimers();
    runTokenRef.current++;
    personasRef.current = [];
    edgesRef.current = [];
    setGraph({ personas: [], edges: [] });
    removeVideo();
    setNiche("");
    setDemographic("");
    setContentDescription("");
    setRunState("idle");
    setStatusText("");
    setRunId("");
    setVerdict(null);
    setBreakdown(null);
    setAnalysis(null);
    setSelectedPersona(null);
    selectedIdRef.current = null;
    setCurrentRound(0);
    drawEmptyCanvas();
  };

  useEffect(() => {
    // Re-theme the placeholder only; an existing graph is redrawn by the effect below.
    if (!personasRef.current.length) drawEmptyCanvas();
  }, [drawEmptyCanvas]);

  useEffect(() => {
    if (currentRound > 0) redraw(currentRound, selectedIdRef.current, hoveredIdRef.current);
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
    const text = buildSummaryText(verdict, analysis, runId, population);
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
    downloadJSON(
      { runId, niche, demographic, hook: contentDescription, verdict, analysis, breakdown, personas: personasRef.current },
      `viralyst-${runId || Date.now()}.json`
    );
  };

  // The backdrop's crowd takes on the result's reaction mix.
  const mix = useMemo(() => {
    if (!breakdown) return null;
    const total = breakdown.shares + breakdown.comments + breakdown.likes + breakdown.scrolls || 1;
    return { share: breakdown.shares / total, comment: breakdown.comments / total, like: breakdown.likes / total };
  }, [breakdown]);
  const mood = running ? "running" : runState === "done" ? "done" : "idle";

  return (
    <div className="grain relative min-h-screen overflow-x-clip">
      <Suspense fallback={null}>
        <Backdrop3D dark={darkMode} mood={mood} mix={mix} />
      </Suspense>

      <div className="relative z-10">
        <Header darkMode={darkMode} setDarkMode={setDarkMode} canReset={runState !== "idle" || !!niche || !!contentDescription} resetAll={resetAll} />
        <Hero />

        <main
          id="simulate"
          className="mx-auto grid max-w-[1680px] scroll-mt-16 grid-cols-1 gap-6 px-4 pb-24 pt-6 sm:px-8 lg:grid-cols-[400px_minmax(0,1fr)] lg:items-start"
        >
          <SetupPanel
            applyExample={applyExample}
            videoPreview={videoPreview}
            fileName={fileName}
            fileInputRef={fileInputRef}
            handleFile={handleFile}
            removeVideo={removeVideo}
            niche={niche}
            setNiche={setNiche}
            demographic={demographic}
            setDemographic={setDemographic}
            contentDescription={contentDescription}
            setContentDescription={setContentDescription}
            population={population}
            setPopulation={setPopulation}
            boost={boost}
            setBoost={setBoost}
            running={running}
            runSim={runSim}
          />

          <div className="min-w-0 space-y-6">
            <ResultsPanel
              runState={runState}
              statusText={statusText}
              verdict={verdict}
              breakdown={breakdown}
              analysis={analysis}
              population={population}
              copied={copied}
              copySummary={copySummary}
              exportJSON={exportJSON}
            />

            <SpreadGraph
              darkMode={darkMode}
              currentRound={currentRound}
              canvasRef={canvasRef}
              handleCanvasClick={handleCanvasClick}
              handleCanvasMouseMove={handleCanvasMouseMove}
              handleCanvasLeave={handleCanvasLeave}
              hoveredPersona={hoveredPersona}
              selectedPersona={selectedPersona}
              selectPersona={selectPersona}
              closePersona={closePersona}
              hasResult={runState === "done"}
              replay={replay}
              showRound={showRound}
              view={view}
              setView={setView}
              personas={graph.personas}
              edges={graph.edges}
              population={population}
            />
          </div>
        </main>

        <footer className="border-t border-ink/10 px-4 py-8 text-xs text-ink/50 sm:px-8">
          <div className="mx-auto flex max-w-[1680px] flex-wrap justify-between gap-2">
            <span className="font-display text-sm font-bold text-ink/70">Viralyst</span>
            <span>Simulated audiences are a model, not a guarantee — use the verdict to compare drafts.</span>
          </div>
        </footer>
      </div>
    </div>
  );
}
