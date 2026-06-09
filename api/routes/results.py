# routes/results.py
# Handles polling — frontend asks "is my job done yet?"
# Also returns the full result once the Go worker completes it.

from fastapi import APIRouter, HTTPException
import redis
import json

router = APIRouter()

r = redis.Redis(host="localhost", port=6379, decode_responses=True)

@router.get("/status/{job_id}")
def get_status(job_id: str):
    """
    Frontend polls this every 2 seconds after hitting RUN.
    Returns current status: pending / running / complete / failed
    """
    job_data = r.hgetall(f"job:{job_id}")

    # hgetall returns empty dict if key doesn't exist
    if not job_data:
        raise HTTPException(status_code=404, detail="Job not found")

    return {"job_id": job_id, "status": job_data.get("status", "pending")}


@router.get("/results/{job_id}")
def get_results(job_id: str):
    """
    Called by frontend once status is 'complete'.
    Returns the full simulation result from Redis.
    Go worker writes results here after processing.
    """
    result_data = r.get(f"result:{job_id}")

    if not result_data:
        raise HTTPException(status_code=404, detail="Results not ready yet")

    # Parse the JSON string back into a Python dict
    return json.loads(result_data)