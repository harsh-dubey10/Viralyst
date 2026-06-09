# routes/simulate.py
# Handles the /api/simulate endpoint.
# React calls this when the user hits RUN.
# We create a job, push it to Redis, return the job_id to frontend.

from fastapi import APIRouter, UploadFile, File, Form
from typing import Optional
import redis
import json
import uuid
import os

from models import SimulationJob, SimulationResponse, SimulationStatus

# APIRouter = a mini-app that groups related routes
# We register this in main.py
router = APIRouter()

# Connect to Redis
# Redis is running locally for now — will be a Docker service later
r = redis.Redis(host="localhost", port=6379, decode_responses=True)

@router.post("/simulate", response_model=SimulationResponse)
async def run_simulation(
    demographic: str = Form(...),           # Form(...) means required form field
    population: int = Form(100),
    boost: int = Form(6),
    content_description: Optional[str] = Form(None),
    video: Optional[UploadFile] = File(None),  # Optional video file upload
):
    """
    Called by React when user hits RUN.
    1. Generates a unique job ID
    2. Saves video file if uploaded
    3. Creates a SimulationJob object
    4. Pushes it to Redis queue
    5. Returns job_id so frontend can poll for results
    """

    # Generate a unique job ID — uuid4 gives a random UUID like 'a3f2b1c4-...'
    job_id = str(uuid.uuid4())

    # Handle video upload
    video_filename = None
    if video:
        # Save uploaded video to a temp directory
        os.makedirs("uploads", exist_ok=True)
        video_filename = f"{job_id}_{video.filename}"
        video_path = f"uploads/{video_filename}"
        with open(video_path, "wb") as f:
            content = await video.read()
            f.write(content)

    # Create the job object
    job = SimulationJob(
        job_id=job_id,
        demographic=demographic,
        population=population,
        boost=boost,
        content_description=content_description,
        video_filename=video_filename,
        status=SimulationStatus.PENDING,
    )

    # Push job to Redis queue
    # "simulation_queue" is the queue name Go workers will listen to
    # json.dumps converts the Python dict to a JSON string for Redis storage
    r.lpush("simulation_queue", job.model_dump_json())

    # Also store job status in Redis so frontend can poll it
    # We store it as a hash with the job_id as key
    r.hset(f"job:{job_id}", mapping={"status": "pending", "job_id": job_id})

    # Set expiry — jobs expire after 1 hour to keep Redis clean
    r.expire(f"job:{job_id}", 3600)

    return SimulationResponse(
        job_id=job_id,
        status=SimulationStatus.PENDING,
        message=f"Simulation queued with {population} agents",
    )