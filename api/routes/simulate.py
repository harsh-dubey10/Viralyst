# routes/simulate.py
# Handles the /api/simulate endpoint.
# Receives simulation request, validates payload, stores video (if any), pushes job to Redis.

from fastapi import APIRouter, UploadFile, File, Form, HTTPException
from typing import Optional
import json
import uuid
import os
import logging

from models import SimulationJob, SimulationResponse, SimulationStatus
from redis_client import redis_client, memory_jobs, memory_queue

router = APIRouter()
logger = logging.getLogger("viralyst.simulate")

MAX_FILE_SIZE = 50 * 1024 * 1024  # 50 MB
ALLOWED_EXTENSIONS = {".mp4", ".mov", ".webm", ".mkv", ".avi", ".m4v"}

@router.post("/simulate", response_model=SimulationResponse)
async def run_simulation(
    demographic: str = Form(...),
    population: int = Form(100),
    boost: int = Form(6),
    content_description: Optional[str] = Form(None),
    video: Optional[UploadFile] = File(None),
):
    job_id = str(uuid.uuid4())

    video_filename = None
    if video and video.filename:
        ext = os.path.splitext(video.filename)[1].lower()
        if ext not in ALLOWED_EXTENSIONS:
            raise HTTPException(
                status_code=400,
                detail=f"Unsupported file type '{ext}'. Allowed extensions: {', '.join(sorted(ALLOWED_EXTENSIONS))}"
            )
        
        os.makedirs("uploads", exist_ok=True)
        safe_name = os.path.basename(video.filename).replace(" ", "_")
        video_filename = f"{job_id}_{safe_name}"
        video_path = os.path.join("uploads", video_filename)

        content = await video.read()
        if len(content) > MAX_FILE_SIZE:
            raise HTTPException(
                status_code=400,
                detail="Uploaded file exceeds the 50MB size limit."
            )

        with open(video_path, "wb") as f:
            f.write(content)

    job = SimulationJob(
        job_id=job_id,
        demographic=demographic,
        population=population,
        boost=boost,
        content_description=content_description,
        video_filename=video_filename,
        status=SimulationStatus.PENDING,
    )

    job_data_json = job.model_dump_json()

    pushed_to_redis = False
    if redis_client is not None:
        try:
            redis_client.lpush("simulation_queue", job_data_json)
            redis_client.hset(f"job:{job_id}", mapping={"status": "pending", "job_id": job_id})
            redis_client.expire(f"job:{job_id}", 3600)
            pushed_to_redis = True
            logger.info(f"Successfully queued job {job_id} to Redis simulation_queue")
        except Exception as e:
            logger.error(f"Failed to push job {job_id} to Redis: {e}")

    if not pushed_to_redis:
        memory_jobs[job_id] = {"status": "pending", "job_id": job_id}
        memory_queue.append(job_data_json)
        logger.info(f"Queued job {job_id} in memory fallback queue")

    return SimulationResponse(
        job_id=job_id,
        status=SimulationStatus.PENDING,
        message=f"Simulation queued with {population} agents",
    )
