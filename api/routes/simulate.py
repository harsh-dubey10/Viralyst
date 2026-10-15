# routes/simulate.py
# Handles the /api/simulate endpoint.
# Receives simulation request, validates payload, stores video (if any), pushes job to Redis.

from fastapi import APIRouter, UploadFile, File, Form, HTTPException
from typing import Optional
import uuid
import os
import re
import time
import logging
import redis

from models import SimulationJob, SimulationResponse, SimulationStatus
from redis_client import redis_client, JOB_TTL_SECONDS

router = APIRouter()
logger = logging.getLogger("viralyst.simulate")

UPLOAD_DIR = "uploads"
MAX_FILE_SIZE = 50 * 1024 * 1024  # 50 MB
CHUNK_SIZE = 1024 * 1024  # 1 MB
ALLOWED_EXTENSIONS = {".mp4", ".mov", ".webm", ".mkv", ".avi", ".m4v"}


def purge_expired_uploads() -> None:
    """Delete uploads older than the job TTL — their jobs and results have already expired."""
    cutoff = time.time() - JOB_TTL_SECONDS
    try:
        entries = list(os.scandir(UPLOAD_DIR))
    except FileNotFoundError:
        return
    for entry in entries:
        if entry.name.startswith(".") or not entry.is_file():
            continue
        try:
            if entry.stat().st_mtime < cutoff:
                os.remove(entry.path)
        except OSError as e:
            logger.warning(f"Could not remove expired upload {entry.name}: {e}")


async def save_upload(video: UploadFile, job_id: str) -> str:
    """Stream the upload to disk in chunks, enforcing the size limit without buffering it all in memory."""
    ext = os.path.splitext(video.filename)[1].lower()
    if ext not in ALLOWED_EXTENSIONS:
        raise HTTPException(
            status_code=400,
            detail=f"Unsupported file type '{ext}'. Allowed extensions: {', '.join(sorted(ALLOWED_EXTENSIONS))}",
        )

    os.makedirs(UPLOAD_DIR, exist_ok=True)
    safe_name = re.sub(r"[^A-Za-z0-9._-]", "_", os.path.basename(video.filename))
    video_filename = f"{job_id}_{safe_name}"
    video_path = os.path.join(UPLOAD_DIR, video_filename)

    written = 0
    try:
        with open(video_path, "wb") as f:
            while chunk := await video.read(CHUNK_SIZE):
                written += len(chunk)
                if written > MAX_FILE_SIZE:
                    raise HTTPException(status_code=413, detail="Uploaded file exceeds the 50MB size limit.")
                f.write(chunk)
    except BaseException:
        if os.path.exists(video_path):
            os.remove(video_path)
        raise

    return video_filename


@router.post("/simulate", response_model=SimulationResponse)
async def run_simulation(
    demographic: str = Form("", max_length=500),
    niche: str = Form("", max_length=80),
    population: int = Form(100, ge=50, le=250),
    boost: int = Form(6, ge=1, le=10),
    content_description: Optional[str] = Form(None, max_length=2000),
    video: Optional[UploadFile] = File(None),
):
    if not demographic.strip() and not niche.strip():
        raise HTTPException(status_code=422, detail="Pick a niche or describe your target audience.")

    # Fail fast before accepting an upload if the worker queue is unreachable.
    try:
        redis_client.ping()
    except redis.RedisError:
        raise HTTPException(status_code=503, detail="Job queue (Redis) is unavailable — start redis-server and retry.")

    job_id = str(uuid.uuid4())

    purge_expired_uploads()

    video_filename = None
    if video and video.filename:
        video_filename = await save_upload(video, job_id)

    job = SimulationJob(
        job_id=job_id,
        demographic=demographic.strip(),
        niche=niche.strip(),
        population=population,
        boost=boost,
        content_description=content_description.strip() if content_description else None,
        video_filename=video_filename,
        status=SimulationStatus.PENDING,
    )

    try:
        # Create the status hash before queueing so the worker never races ahead of it.
        pipe = redis_client.pipeline()
        pipe.hset(f"job:{job_id}", mapping={"status": "pending", "job_id": job_id})
        pipe.expire(f"job:{job_id}", JOB_TTL_SECONDS)
        pipe.lpush("simulation_queue", job.model_dump_json())
        pipe.execute()
    except redis.RedisError as e:
        logger.error(f"Failed to queue job {job_id}: {e}")
        if video_filename:  # no job will ever read it
            try:
                os.remove(os.path.join(UPLOAD_DIR, video_filename))
            except OSError:
                pass
        raise HTTPException(status_code=503, detail="Failed to queue simulation job")

    logger.info(f"Queued job {job_id} to simulation_queue")
    return SimulationResponse(
        job_id=job_id,
        status=SimulationStatus.PENDING,
        message=f"Simulation queued with {population} agents",
    )
