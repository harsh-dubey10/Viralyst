<div align="center">

# 🚀 Viralyst

### **Multi-Agent Synthetic Audience**

*Simulate social media virality, audience retention, and network cascade dynamics before publishing.*

[![Go Version](https://img.shields.io/badge/Go-1.25%2B-00ADD8?style=for-the-badge&logo=go&logoColor=white)](https://golang.org/)
[![FastAPI](https://img.shields.io/badge/FastAPI-0.100%2B-009688?style=for-the-badge&logo=fastapi&logoColor=white)](https://fastapi.tiangolo.com/)
[![React](https://img.shields.io/badge/React-18%2F19-61DAFB?style=for-the-badge&logo=react&logoColor=black)](https://reactjs.org/)
[![TypeScript](https://img.shields.io/badge/TypeScript-5.0-3178C6?style=for-the-badge&logo=typescript&logoColor=white)](https://www.typescriptlang.org/)
[![Tailwind CSS](https://img.shields.io/badge/Tailwind_CSS-3.4-38B2AC?style=for-the-badge&logo=tailwind-css&logoColor=white)](https://tailwindcss.com/)
[![Redis](https://img.shields.io/badge/Redis-Queue%20%26%20State-DC382D?style=for-the-badge&logo=redis&logoColor=white)](https://redis.io/)
[![Groq](https://img.shields.io/badge/Groq-gpt--oss--20b-F05A28?style=for-the-badge)](https://groq.com/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg?style=for-the-badge)](https://opensource.org/licenses/MIT)

[Features](#-key-features) • [Architecture](#-system-architecture) • [How It Works](#-simulation-mechanics--diffusion-dynamics) • [Quickstart](#-quickstart--installation) • [API Reference](#-api-reference) • [Author](#-author)

---

</div>

## 💡 Why Viralyst?

Traditional social media analytics are **strictly post-mortem**—you only discover whether a video hooks viewers, flops, or gets trapped in an echo chamber **after** you have burned your distribution, ad spend, and algorithmic goodwill.

**Viralyst flips the script:**

> **What if you could test your video hook against hundreds of AI personas with diverse demographics, realistic attention spans, and genuine social media skepticism before ever hitting "Publish"?**

Viralyst generates a synthetic audience of autonomous AI agents across diverse demographics, psychographics, and follower tiers. It simulates their split-second decisions (watch retention, like, comment, share, or skip) and models how content cascades through direct network sharing and algorithmic recommendation pushes across 3 diffusion rounds.

```
+------------------+      +-----------------------+      +---------------------------+
|  Your Video Hook | ---> | 100+ Synthetic Agents | ---> | Interactive Spread Graph  |
|  & Target Demo   |      |   (Groq LLM or offline)  |      | & Virality Verdict (0-100)|
+------------------+      +-----------------------+      +---------------------------+
```

---

## ✨ Key Features

### 🧠 1. Realistic Multi-Agent Persona Simulation
* **Psychological Realism:** Real users doomscroll with short attention spans and swipe past 75%–85% of content in under 3 seconds. Viralyst agents model authentic skepticism rather than sycophantic agreement.
* **19 Behavioral Archetypes:** Simulates *sharers, lurkers, skeptics, critics, influencers, early adopters, analytical thinkers, trend followers, mentors,* and more.
* **Granular Persona Identities:** Each agent has realistic global locations (US, UK, India, Germany, Brazil, Japan, etc.), culturally accurate names, matched genders, follower counts (50 to 10,000+), and multi-interest vectors.
* **LLM-Powered Reasoning:** When enabled, personas articulate **first-person rationale** explaining why they shared, commented, liked, or swiped past.

### 🌐 2. Dual-Cohort Spread Canvas Visualizer
* **High-Performance HTML5 Canvas:** Renders hundreds of interactive nodes, curved Bezier cascades, and algorithmic exposure vectors at 60 FPS.
* **Target vs. Out-of-Target Cohorts:** Visualizes in-target demographic engagement vs. algorithmic out-of-target crossover side-by-side.
* **3-Round Live Propagation:**
  * **Round 1 (Initial Hook):** Direct reaction from the initial seed audience based on content quality and hook strength.
  * **Round 2 (Direct Share Cascade):** Network expansion triggered by organic shares, weighted by follower reach.
  * **Round 3 (Algorithmic Push):** Recommendation system expansion into out-of-target cohorts driven by high engagement velocity and interest clustering.
* **Interactive Inspection Drawer:** Click any persona node to view their full profile, watch percentage, engagement decision, and LLM reasoning.
* **Replay & Scrub Bar:** Replay the multi-round cascade step-by-step or scrub back and forth.

### ⚡ 3. Dual Groq Multi-Key Load Balancing Engine
* **Concurrent Go Worker Pool:** High-throughput Go engine utilizing goroutines and semaphores for lightning-fast parallel persona evaluations.
* **Multi-Key Round-Robin:** Automatically discovers `GROQ_API_KEY`, `GROQ_API_KEY_1`...`GROQ_API_KEY_10` or comma-separated keys, distributing requests to bypass single-key rate limits.
* **Automatic 429 Rate-Limit Failover:** Seamlessly fails over to alternate keys upon hitting rate limits, backed by an intelligent heuristic fallback engine for uninterrupted simulations.

### 📊 4. Executive Virality Verdict & Deep Analytics
* **Virality Index (0–100):** 60% engagement quality (shares $3\times$, comments $2\times$, likes $1\times$) + 40% spread beyond the seed audience.
* **Core Predictive KPIs:** Total Reach, In/Out Demo Split %, Organic Share Rate %, Algorithm Score, and Non-Skip Engagement Rate %.
* **Qualitative Verdict Engine:** Categorizes outcomes into actionable verdicts:
  * ↔️ **Audience Mismatch:** The hook's topics don't match what the target audience cares about.
  * 🏆 **Viral Hit:** Strong in-demo spread with organic cross-demo viral breakout.
  * 🎯 **Niche Hit:** Deep engagement within target niche, limited broader crossover.
  * 🔁 **Echo Chamber:** High retention inside target group, fails algorithmic push.
  * ⚠️ **Weak Hook:** 60%+ of viewers who care about the topic still scrolled past in the first 3 seconds.
  * 📉 **Weak Signal:** Low engagement and share rate across all cohorts.
* **Interactive Charts:** Recharts-powered Audience Engagement Funnels and Cohort Distribution Pie Charts.
* **Export & Share:** 1-click JSON export and clipboard summary report generation.

---

## 🏗️ System Architecture

Viralyst is built as an asynchronous, event-driven distributed system designed for low latency and high concurrency:

```mermaid
flowchart TD
    subgraph Client["Frontend (React 18 + TypeScript + Tailwind)"]
        UI["UI / Campaign Setup / Video Upload"]
        Canvas["HTML5 Canvas Spread Visualizer"]
        Charts["Recharts Analytics & KPI Dashboard"]
    end

    subgraph Gateway["API Gateway (FastAPI)"]
        API["FastAPI Async REST API (:8000)"]
        Uploads["Video Media Storage (50MB Max)"]
    end

    subgraph Broker["Message Broker & State Store"]
        Queue[("Redis: simulation_queue (LPUSH / BLPOP)")]
        State[("Redis: job:{id} & result:{id} State Store")]
    end

    subgraph Engine["Worker Engine (Go Concurrent Daemon)"]
        Worker["Go Worker Pool (Goroutines + Semaphore)"]
        PersonaGen["Demographic Persona Generator"]
        SpreadGraph["3-Round Network Diffusion Engine"]
        VerdictCalc["Virality Verdict & KPI Aggregator"]
    end

    subgraph AI["AI Layer & Multi-Key LLM Rotation"]
        GroqPool["Multi-Key Groq LLM (GROQ_MODEL) Load Balancer"]
        HeuristicEngine["Stochastic Heuristic Fallback Engine"]
    end

    UI -->|"POST /api/simulate"| API
    API -->|"Store media"| Uploads
    API -->|"Push Job"| Queue
    API -->|"Init Pending"| State

    Worker -->|"BLPop (5s timeout)"| Queue
    Worker -->|"Generate Personas"| PersonaGen
    Worker -->|"Evaluate Personas"| GroqPool
    GroqPool -.->|"On 429 / Rate Limit"| HeuristicEngine
    Worker -->|"Compute Spread"| SpreadGraph
    Worker -->|"Aggregate KPIs"| VerdictCalc
    Worker -->|"Set Complete & Results"| State

    UI -->|"Poll GET /api/status/{id}"| API
    API -->|"Read Status"| State
    UI -->|"GET /api/results/{id}"| API
    API -->|"Fetch Payload"| State
    API --> Canvas
    API --> Charts
```

---

## 🔬 Simulation Mechanics & Diffusion Dynamics

### 1. Reading the inputs (`worker/analysis.go`)
Both free-text inputs are parsed into structured signals before any persona is created:

* **Audience** → interest topics (from a 44-topic lexicon, whole-word matching), age range (`Gen Z`, `millennials`, `25-34`, `40+` …) and regions (`India`, `US`, `Europe` …).
* **Hook** → content topics, plus a **hook score** (0–100) from concrete signals: specific numbers/results, curiosity or pattern-interrupt language, a clear payoff (steps, framework, tested), direct address, a short opening line — and penalties for salesy CTAs, shouting, or thin/vague text.
* **Audience fit** = share of the audience's interests the hook actually talks about.
* **Video** (`worker/video.go`, needs `ffmpeg` and a Groq key) → ffmpeg pulls the audio and one frame per second from the first 3 seconds, tiling the frames into a single small image to stay within Groq's free-tier token limits. Groq Whisper transcribes the audio (ignoring its "[Music]"-style non-speech outputs) and a Groq vision model returns the on-screen text, a scene description and what the video is about. Topics and hook strength are scored from the speech, on-screen text and subject; the scene is passed to the LLM personas but not scored, since props and settings aren't the topic. Everything extracted appears in the UI under **Read from your video**. If the video can't be analysed, the reason is listed under the recommendations and the text description is used on its own.

### 2. Personas are built from the audience
In-target personas (50%) get interests drawn from the audience's topics and ages/regions from the parsed ranges; out-of-target personas get interests from *outside* that niche. "In target" therefore means something about who the persona is.

### 3. Decisions depend on relevance × hook
Offline, each persona's reaction is driven by **relevance** (how many of *their* interests the hook covers) and **hook strength**. A great hook matters most to people who already care about the topic:

$$P(\text{stop scrolling}) = 0.03 + \text{relevance}\cdot(0.20 + 0.65\cdot\text{hook}) + 0.10\cdot\text{hook}$$

Share / comment / like odds then scale with $v = 0.55\cdot\text{relevance} + 0.45\cdot\text{hook}$; behavior archetype and follower count only nudge them. With a Groq key, an LLM plays each persona instead (it is told the persona's interests, never whether it is "in target").

### 4. The 3 diffusion rounds
* **Round 1 — Seed:** $\text{Boost} \times 4$ personas (≈75% in-target) see the post.
* **Round 2 — Direct shares:** each sharer reaches 1–4 people (more with more followers), chosen by **homophily** — people share with others who have similar interests.
* **Round 3 — Algorithmic push:** the platform recommends the post to a share of unreached users proportional to engagement so far, picking those whose interests best match the content:

$$\text{push rate} = \text{clamp}\big((E_w - 0.08)\times 2.5,\ 0,\ 0.6\big), \quad E_w = \frac{3S + 2C + L}{3\cdot N_{\text{reached}}}$$

### 5. Scores and verdict
$$\text{Virality} = 100\cdot\Big(0.6\cdot\min\big(1, \tfrac{E_w}{0.4}\big) + 0.4\cdot\frac{N_{\text{reached}} - N_{\text{seed}}}{N_{\text{pop}} - N_{\text{seed}}}\Big)$$

| Verdict | Rule (checked in order) |
| :--- | :--- |
| ↔️ **Audience mismatch** | Hook topics don't overlap the audience's interests and < 25% of in-target viewers engaged |
| ⚠️ **Weak hook** | ≥ 60% of seed viewers *for whom the topic is relevant* still skipped |
| 🏆 **Viral hit** | Virality ≥ 60, share rate ≥ 10%, out-of-target engagement ≥ 30% |
| 🎯 **Niche hit** | In-target engagement ≥ 40% and share rate ≥ 6% |
| 🔁 **Echo chamber** | In-target engagement ≥ 35% but few shares |
| 📉 **Weak signal** | Anything else |

Every result includes an `analysis` block (detected topics, fit, hook score and signals, in/out engagement, recommendations), shown in the UI under **Why this verdict**.

### 6. Confidence
Offline runs are cheap (~2 ms), so the worker simulates **25 replicates** with different seeds, reports the most common verdict with its agreement rate as **confidence**, and displays a replicate that produced it. LLM runs are simulated once (too costly to repeat).

### 7. Validating the model
`worker/scenario_test.go` runs fixed scenarios (strong vs weak hook, right vs wrong audience, spam, empty, broad appeal) 10–30 times each and asserts the expected verdict and ranking. Print the table with `go test -run ScenarioReport -v`. A live LLM run is opt-in: `VIRALYST_LLM_TEST=1 go test -run LLMScenarios -v`.

Those scenarios only check the model behaves as designed. To check it against **reality**, add real posts with known outcomes to `eval/posts.csv` (column guide is at the top of the file) and run:

```bash
cd worker
go run . -eval ../eval/posts.csv -offline   # drop -offline to use Groq personas (costs tokens)
```

It simulates each post and reports the **Spearman rank correlation** between the virality score and your real outcome, and whether it is distinguishable from chance at that sample size. Aim for 30+ posts that include flops.

> **Caveat:** these coefficients encode plausible assumptions; until `eval/posts.csv` has real data behind it, they are not validated against real post performance, so treat scores as *relative* (compare hooks against each other), not as absolute view predictions.

---

## 📁 Repository Structure

```
Viralyst/
├── api/                        # FastAPI REST API Backend
│   ├── routes/
│   │   ├── simulate.py         # /api/simulate endpoint (validation & queueing)
│   │   └── results.py          # /api/status & /api/results endpoints
│   ├── main.py                 # FastAPI app entrypoint & CORS middleware
│   ├── models.py               # Pydantic data schemas
│   ├── redis_client.py         # Shared Redis client (job queue & state)
│   ├── requirements.txt        # Python dependencies
│   ├── Dockerfile              # Backend container definition
│   └── .env.example            # API environment variables
│
├── worker/                     # High-Performance Go Simulation Engine
│   ├── main.go                 # Worker daemon entrypoint & config loading
│   ├── worker.go               # Redis consumer & multi-round spread graph
│   ├── analysis.go             # Parses audience & hook into topics, fit, hook score
│   ├── persona.go              # Persona generator (demographics & archetypes)
│   ├── groq.go                 # Groq LLM client (retry/backoff) & offline decision model
│   ├── video.go                # Video hook extraction (ffmpeg → Whisper + vision model)
│   ├── eval.go                 # `-eval`: rank correlation against real post outcomes
│   ├── types.go                # Go struct definitions
│   ├── go.mod / go.sum         # Go module dependencies
│   ├── Dockerfile              # Go worker container definition
│   └── .env.example            # Worker environment variables
│
├── frontend/                   # React + TypeScript Web Application
│   ├── src/
│   │   ├── App.tsx             # Top-level state & simulation flow
│   │   ├── components/         # Header, setup form, spread graph, results, persona card
│   │   ├── lib/                # Canvas drawing & report export helpers
│   │   ├── api.ts / types.ts   # API client & shared types
│   │   ├── index.tsx           # React DOM root
│   │   └── index.css           # Tailwind styling & animations
│   ├── public/                 # Static assets
│   ├── tailwind.config.js      # Tailwind CSS configuration
│   ├── tsconfig.json           # TypeScript configuration
│   ├── Dockerfile              # Frontend container definition
│   └── package.json            # React & NPM dependencies
│
├── eval/posts.csv              # Real posts with known outcomes (you fill this in)
├── docker-compose.yml          # Multi-container orchestration (Redis, API, Worker, UI)
├── run.sh                      # 1-command startup script (launches all services)
├── package.json                # Root metadata & start scripts
├── LICENSE                     # MIT License
└── README.md                   # Project documentation
```

---

## ⚡ Quickstart & Installation

### Prerequisites
* **Node.js** (v18+ recommended)
* **Python** (v3.10+ recommended)
* **Go** (v1.25+ required)
* **ffmpeg** (*Optional* — needed for video analysis; `brew install ffmpeg`. Docker images include it)
* **Redis** (*Required* — it is the job queue between the API and the Go worker; the API returns `503` while it is unreachable)

---

### Option 1: 1-Command Startup (Recommended)

Clone the repository and run the automated startup script:

```bash
# 1. Clone repository
git clone https://github.com/harsh-dubey10/Viralyst.git
cd Viralyst

# 2. Configure environment variables (optional for Groq LLM)
cp .env.example .env
# Add your GROQ_API_KEY inside .env

# 3. Install frontend & Python dependencies
cd frontend && npm install && cd ..
cd api && python3 -m pip install -r requirements.txt && cd ..

# 4. Launch all services with a single command
./run.sh
```

Viralyst will start Redis (must be installed), the FastAPI API on `http://localhost:8000`, the Go Worker daemon, and the React UI on `http://localhost:3000`.

---

### Option 2: Docker Compose

Run the entire distributed stack (Redis + API + Go Worker + React UI) in isolated containers:

```bash
# 1. Clone and enter repo
git clone https://github.com/harsh-dubey10/Viralyst.git
cd Viralyst

# 2. Copy and configure .env
cp .env.example .env

# 3. Launch stack
docker compose up --build
```

Access the UI at `http://localhost:3000` and the API at `http://localhost:8000`.

---

### Option 3: Manual Step-by-Step Setup

<details>
<summary><b>Click to expand step-by-step instructions</b></summary>

#### 1. Start Redis
```bash
redis-server
```

#### 2. Start FastAPI Backend
```bash
cd api
python3 -m venv venv
source venv/bin/activate
pip install -r requirements.txt
uvicorn main:app --host 0.0.0.0 --port 8000 --reload
```

#### 3. Start Go Worker Engine
```bash
cd worker
go run .
```

#### 4. Start React Frontend
```bash
cd frontend
npm install
npm start
```

</details>

---

## 🔑 Environment Variables & Multi-Key Groq Setup

Viralyst supports **multi-key load balancing** across Groq API keys for failover. Note: keys from the **same Groq organization share one rate limit** (e.g. 8k tokens/min on the free tier), so extra keys only add capacity if they come from different organizations. When every key is rate limited the worker waits as long as Groq asks (`Retry-After`) and retries.

| Variable | Description | Default / Example |
| :--- | :--- | :--- |
| `REDIS_URL` | Connection URL for Redis | `redis://localhost:6379` |
| `WORKER_CONCURRENCY` | Maximum concurrent persona evaluations (LLM calls) | `10` |
| `CORS_ORIGINS` | API: comma-separated allowed frontend origins | `http://localhost:3000` |
| `REACT_APP_API_URL` | Frontend (build time): API base URL | `http://localhost:8000` |
| `GROQ_API_KEY` | Primary Groq API Key | `gsk_...` |
| `GROQ_API_KEY_1`..`_10` | Additional numbered Groq keys for automatic rotation | `gsk_...` |
| `GROQ_API_KEYS` | Comma-separated list of Groq keys | `gsk_1...,gsk_2...` |
| `GROQ_MODEL` | Groq model for persona decisions | `openai/gpt-oss-20b` |
| `GROQ_VISION_MODEL` | Groq model that describes the video's opening frames | `qwen/qwen3.8-27b` |
| `GROQ_TRANSCRIBE_MODEL` | Groq model that transcribes the video's opening audio | `whisper-large-v3-turbo` |
| `UPLOAD_DIR` | Worker: where the API saves uploads | `../api/uploads` (Docker: `/app/uploads`) |

> [!TIP]
> If no Groq API keys are provided, Viralyst's **Stochastic Heuristic Fallback Engine** automatically handles persona evaluations using demographic, behavioral, and attention-decay algorithms.

---

## 📡 API Reference

### 1. `POST /api/simulate`
Queues a new social media virality simulation.

**Request (`multipart/form-data`):**
* `demographic` *(string, required)*: Target audience description (e.g. `"Gen Z AI developers"`).
* `population` *(integer, optional, default: 100)*: Number of personas to simulate (50–250).
* `boost` *(integer, optional, default: 6)*: Initial seed amplification multiplier (1–10).
* `content_description` *(string, optional)*: Hook script, text description, or CTA.
* `video` *(file, optional)*: Uploaded video file (MP4, MOV, WEBM, MKV, AVI, max 50MB).

**Response (`application/json`):**
```json
{
  "job_id": "4b92ec84-90ab-48cb-96d5-a921c19c557f",
  "status": "pending",
  "message": "Simulation queued with 100 agents"
}
```

---

### 2. `GET /api/status/{job_id}`
Polls the execution status of a simulation job.

**Response (`application/json`):**
```json
{
  "job_id": "4b92ec84-90ab-48cb-96d5-a921c19c557f",
  "status": "running" // "pending" | "running" | "complete" | "failed"
}
```

---

### 3. `GET /api/results/{job_id}`
Fetches the complete simulation payload, graph edges, persona inspection data, and verdict.

**Sample Response (`application/json`):**
```json
{
  "job_id": "4b92ec84-90ab-48cb-96d5-a921c19c557f",
  "status": "complete",
  "virality": 78,
  "algo_score": 93,
  "engagement": 82,
  "breakdown": {
    "likes": 34,
    "shares": 18,
    "comments": 12,
    "scrolls": 14
  },
  "verdict": {
    "head": "Viral hit.",
    "sub": "Strong in-demo spread with organic cross-demo viral spread. High viral coefficient.",
    "verdict": "Breakout potential",
    "in_pct": 58,
    "share_rate": 23
  },
  "personas": [
    {
      "persona_id": 1,
      "name": "Marcus Webb",
      "age": 24,
      "gender": "male",
      "location": "New York, US",
      "followers": 4200,
      "interests": ["B2B SaaS", "growth hacking"],
      "behavior": "sharer",
      "in_target": true,
      "shown": true,
      "action": "share",
      "watch": 92,
      "round": 1,
      "why": "This is exactly the distribution signal my team was discussing yesterday. High-value framework."
    }
  ],
  "edges": [
    { "from": 1, "to": 14, "type": "direct", "round": 2 },
    { "from": 1, "to": 38, "type": "algo", "round": 3 }
  ]
}
```

---

## 🛠️ Tech Stack & Dependencies

| Area | Technology | Purpose |
| :--- | :--- | :--- |
| **Frontend** | React 18/19, TypeScript, Tailwind CSS | High-performance interactive UI with Dark/Light theme support |
| **Visualization** | HTML5 2D Canvas, Lucide Icons | 60 FPS graph rendering, bezier curves, halo shaders |
| **Charts** | Recharts | Engagement funnels and demographic cohort distribution charts |
| **API Gateway** | FastAPI, Uvicorn, Pydantic v2 | High-throughput asynchronous REST API & schema validation |
| **Message Broker** | Redis (`go-redis/v9`, `redis-py`) | In-memory job queue (`BLPop`/`LPush`) and state caching |
| **Simulation Engine** | Go (Golang 1.25+) | Concurrent multi-agent worker with goroutines & semaphore pools |
| **LLM Inference** | Groq API (`openai/gpt-oss-20b` by default, `GROQ_MODEL`) | Ultra-fast persona reaction evaluation & reasoning generation |
| **DevOps** | Docker, Docker Compose, Bash | Multi-service containerization and 1-command startup |

---

## 🗺️ Roadmap & Research Directions

- [x] **Multimodal Frame & Audio Analysis:** Opening frames and speech are read with a vision model and Whisper (pace and editing rhythm aren't modelled yet).
- [ ] **Dynamic Social Network Topologies:** Implement Barabási–Albert scale-free and Watts–Strogatz small-world network graphs for organic friend topologies.
- [ ] **Agent Memory & Comment Debates:** Multi-turn interactive discussion threads between synthetic personas reacting to each other's comments.
- [ ] **Distributed Ray Cluster Support:** Scale from 250 personas to 100,000+ agents distributed across Ray cluster worker nodes.

---

## 👨‍💻 Author

**Harsh Vardhan Dubey**
* M.Tech in Computer Science and Engineering, **Indian Institute of Technology Gandhinagar (IITGN)**
* GitHub: [@harsh-dubey10](https://github.com/harsh-dubey10)
* Project Repository: [https://github.com/harsh-dubey10/Viralyst](https://github.com/harsh-dubey10/Viralyst)

---

## 📄 License

This project is licensed under the **MIT License** — see the [LICENSE](LICENSE) file for details.

---

<div align="center">
  <sub>Built with ❤️ by Harsh Vardhan Dubey • Star ⭐ the repository if you find Viralyst inspiring!</sub>
</div>
