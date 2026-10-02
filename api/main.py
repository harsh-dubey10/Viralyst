import os

from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from routes.simulate import router as simulate_router
from routes.results import router as results_router
from redis_client import redis_available

app = FastAPI(title="Viralyst API", version="1.0.0")

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
