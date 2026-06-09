from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from routes.simulate import router as simulate_router
from routes.results import router as results_router

app = FastAPI(title="Viralyst API", version="1.0.0")

app.add_middleware(
    CORSMiddleware,
    allow_origins=["http://localhost:3000"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

app.include_router(simulate_router, prefix="/api")
app.include_router(results_router, prefix="/api")

@app.get("/health")
def health():
    return {"status": "ok", "service": "viralyst-api"}