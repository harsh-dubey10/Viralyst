# models.py
# Pydantic models define the shape of data going in and out of our API.
# Think of them as TypeScript interfaces but for Python — they validate automatically.

from pydantic import BaseModel
from typing import Optional
from enum import Enum

class SimulationStatus(str, Enum):
    # Enum = a fixed set of allowed values
    # Using str means the values serialize to plain strings in JSON
    PENDING = "pending"
    RUNNING = "running"
    COMPLETE = "complete"
    FAILED = "failed"

class SimulationRequest(BaseModel):
    # This is what the frontend sends when hitting POST /api/simulate
    demographic: str                    # e.g. "Gen Z fitness enthusiasts"
    population: int = 100               # default 100 agents
    boost: int = 6                      # seed count
    content_description: Optional[str] = None  # text description if no video

class SimulationJob(BaseModel):
    # This is the job object we push into Redis for Go workers to pick up
    job_id: str
    demographic: str
    population: int
    boost: int
    content_description: Optional[str] = None
    video_filename: Optional[str] = None
    status: SimulationStatus = SimulationStatus.PENDING

class SimulationResponse(BaseModel):
    # What we send back to the frontend after POST /api/simulate
    job_id: str
    status: SimulationStatus
    message: str