# routes/results.py
# Handles polling for simulation status and fetching completed results.

from fastapi import APIRouter, HTTPException
from fastapi.responses import Response
import logging
import redis

from redis_client import redis_client

router = APIRouter()
logger = logging.getLogger("viralyst.results")


@router.get("/status/{job_id}")
def get_status(job_id: str):
    try:
        status = redis_client.hget(f"job:{job_id}", "status")
    except redis.RedisError as e:
        logger.warning(f"Error reading status from Redis for job {job_id}: {e}")
        raise HTTPException(status_code=503, detail="Job store (Redis) is unavailable")

    if not status:
        raise HTTPException(status_code=404, detail="Job not found")

    return {"job_id": job_id, "status": status}


@router.get("/results/{job_id}")
def get_results(job_id: str):
    try:
        result_data = redis_client.get(f"result:{job_id}")
    except redis.RedisError as e:
        logger.warning(f"Error reading results from Redis for job {job_id}: {e}")
        raise HTTPException(status_code=503, detail="Job store (Redis) is unavailable")

    if not result_data:
        raise HTTPException(status_code=404, detail="Results not ready yet")

    # The worker already stores valid JSON — pass it through without re-parsing.
    return Response(content=result_data, media_type="application/json")
