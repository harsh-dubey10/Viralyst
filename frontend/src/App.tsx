import React, { useState, useRef, useEffect, useCallback } from "react";
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
import { ACTION_CONFIG_DARK, ACTION_CONFIG_LIGHT, CampaignPreset } from "./constants";
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
import SetupPanel from "./components/SetupPanel";
import SpreadGraph from "./components/SpreadGraph";
import ResultsPanel from "./components/ResultsPanel";

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

  const applyPreset = (preset: CampaignPreset) => {
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

    const ps = parsePersonas(resultData);

    const { W, H } = canvasRef.current ? canvasSize(canvasRef.current) : { W: 900, H: 540 };
    layoutPersonas(ps, W, H);
    personasRef.current = ps;

    edgesRef.current = parseEdges(resultData);

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
        setVerdict(parseVerdict(resultData));
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

  const closePersona = () => {
    setSelectedPersona(null);
    selectedIdRef.current = null;
    redraw(currentRound, null, hoveredIdRef.current);
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
    const text = buildSummaryText(verdict, runId, population);
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
    downloadJSON(data, `viralyst-simulation-${Date.now()}.json`);
  };


  const actionConfig = darkMode ? ACTION_CONFIG_DARK : ACTION_CONFIG_LIGHT;

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
      <Header
        darkMode={darkMode}
        setDarkMode={setDarkMode}
        analysis={analysis}
        simDone={simDone}
        copied={copied}
        copySummary={copySummary}
        exportJSON={exportJSON}
        resetAll={resetAll}
        runId={runId}
      />

      <main className="max-w-7xl mx-auto px-4 sm:px-6 py-6 space-y-6">
        {/* Preset Quick Fill Tabs + SECTION 01: CAMPAIGN CONFIG */}
        <SetupPanel
          darkMode={darkMode}
          applyPreset={applyPreset}
          videoPreview={videoPreview}
          fileName={fileName}
          fileInputRef={fileInputRef}
          handleFile={handleFile}
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

        {/* SECTION 02: CANVAS SPREAD VISUALIZER */}
        <SpreadGraph
          darkMode={darkMode}
          spreadStatus={spreadStatus}
          currentRound={currentRound}
          canvasRef={canvasRef}
          handleCanvasClick={handleCanvasClick}
          handleCanvasMouseMove={handleCanvasMouseMove}
          hoveredPersona={hoveredPersona}
          selectedPersona={selectedPersona}
          closePersona={closePersona}
          actionConfig={actionConfig}
          simDone={simDone}
          replay={replay}
          scrubRound={scrubRound}
        />

        {/* SECTION 03: VERDICT & METRICS */}
        <ResultsPanel
          darkMode={darkMode}
          verdict={verdict}
          breakdown={breakdown}
          analysis={analysis}
          population={population}
        />
      </main>
    </div>
  );
}
