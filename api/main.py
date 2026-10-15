import os

from fastapi import FastAPI, Request
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import JSONResponse
from routes.simulate import router as simulate_router, MAX_FILE_SIZE
from routes.results import router as results_router
from redis_client import redis_available

app = FastAPI(title="Viralyst API", version="1.0.0")

# Multipart bodies are spooled to disk before the route runs, so the per-chunk check in
# save_upload comes too late to stop a huge upload. Reject by declared size up front
# (the slack covers the form fields and multipart framing). Registered before CORS so
# CORS wraps it and the 413 reaches the browser as a readable error.
MAX_REQUEST_SIZE = MAX_FILE_SIZE + 1024 * 1024


@app.middleware("http")
async def limit_request_size(request: Request, call_next):
    length = request.headers.get("content-length")
    if length and length.isdigit() and int(length) > MAX_REQUEST_SIZE:
        return JSONResponse(status_code=413, content={"detail": "Uploaded file exceeds the 50MB size limit."})
    return await call_next(request)

CORS_ORIGINS = [o.strip() for o in os.getenv("CORS_ORIGINS", "http://localhost:3000").split(",") if o.strip()]

app.add_middleware(
    CORSMiddleware,
    allow_origins=CORS_ORIGINS,
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

app.include_router(simulate_router, prefix="/api")
app.include_router(results_router, prefix="/api")

@app.get("/health")
def health():
    return {"status": "ok", "service": "viralyst-api", "redis": "up" if redis_available() else "down"}
