#!/bin/bash
# Single-command runner for Viralyst

echo "=========================================="
echo "🚀 Starting Viralyst AI Engine..."
echo "=========================================="

REDIS_PID=""
REDIS_STARTED_BY_US=false

# 1. Start Redis if installed
if command -v redis-server > /dev/null 2>&1; then
  if pgrep -x "redis-server" > /dev/null; then
    echo "⚡ Redis server is already active."
  else
    echo "📦 Starting Redis Server..."
    redis-server --daemonize yes --logfile /tmp/viralyst-redis.log 2>/dev/null
    # Give it a moment to start, then capture its PID
    sleep 0.5
    REDIS_PID=$(pgrep -x redis-server | tail -1)
    REDIS_STARTED_BY_US=true
    echo "   Redis started (PID: $REDIS_PID)"
  fi
else
  echo "⚠️  redis-server not found. Simulations will fail (503) until Redis is running."
fi

cleanup() {
  echo ""
  echo "🛑 Shutting down Viralyst processes..."
  # Kill the FastAPI and Go worker background jobs from this shell
  kill $(jobs -p) 2>/dev/null
  # If we started Redis ourselves, shut it down cleanly
  if [ "$REDIS_STARTED_BY_US" = true ] && [ -n "$REDIS_PID" ]; then
    echo "   Stopping Redis (PID: $REDIS_PID)..."
    redis-cli shutdown nosave 2>/dev/null || kill "$REDIS_PID" 2>/dev/null
  fi
  exit 0
}
trap cleanup SIGINT SIGTERM EXIT

# 2. Start FastAPI Backend
echo "🐍 Starting API on http://localhost:8000..."
(cd api && \
  if [ -d "venv" ]; then source venv/bin/activate 2>/dev/null; fi && \
  python3 -m uvicorn main:app --host 0.0.0.0 --port 8000) &

# 3. Start Go Worker Engine
echo "🐹 Starting Go Worker..."
(cd worker && go run .) &

# 4. Wait a moment for backend services to start
sleep 2
echo ""
echo "✅ Services starting:"
echo "   API  → http://localhost:8000"
echo "   Docs → http://localhost:8000/docs"
echo "   UI   → http://localhost:3000"
echo ""

# 5. Start React Frontend (foreground — blocks until Ctrl+C)
echo "⚛️  Starting Frontend on http://localhost:3000..."
(cd frontend && npm start)
