# routes/results.py
# Handles polling for simulation status and fetching completed results.

from fastapi import APIRouter, HTTPException
import json
import logging
from redis_client import redis_client, memory_jobs

router = APIRouter()
logger = logging.getLogger("viralyst.results")

@router.get("/status/{job_id}")
def get_status(job_id: str):
    job_data = None
    if redis_client is not None:
        try:
            job_data = redis_client.hgetall(f"job:{job_id}")
        except Exception as e:
            logger.warning(f"Error reading status from Redis for job {job_id}: {e}")

    if not job_data:
        job_data = memory_jobs.get(job_id)

    if not job_data:
        raise HTTPException(status_code=404, detail="Job not found")

    return {"job_id": job_id, "status": job_data.get("status", "pending")}


@router.get("/results/{job_id}")
def get_results(job_id: str):
    result_data = None
    if redis_client is not None:
        try:
            result_data = redis_client.get(f"result:{job_id}")
        except Exception as e:
            logger.warning(f"Error reading results from Redis for job {job_id}: {e}")

    if not result_data:
        result_data = memory_jobs.get(f"result:{job_id}")

    if not result_data:
        raise HTTPException(status_code=404, detail="Results not ready yet")

    if isinstance(result_data, str):
        try:
            return json.loads(result_data)
        except json.JSONDecodeError:
            return {"raw": result_data}
            
    return result_data
