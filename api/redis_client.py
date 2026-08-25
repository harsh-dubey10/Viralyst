# redis_client.py
# Shared Redis client configuration for Viralyst FastAPI backend.

import os
import redis
import logging

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger("viralyst.redis")

REDIS_URL = os.getenv("REDIS_URL", "redis://localhost:6379/0")

redis_client = None

try:
    if REDIS_URL.startswith("redis://") or REDIS_URL.startswith("rediss://"):
        redis_client = redis.Redis.from_url(
            REDIS_URL,
            decode_responses=True,
            socket_connect_timeout=2,
            socket_timeout=2
        )
    else:
        host = os.getenv("REDIS_HOST", "localhost")
        port = int(os.getenv("REDIS_PORT", "6379"))
        redis_client = redis.Redis(
            host=host,
            port=port,
            decode_responses=True,
            socket_connect_timeout=2,
            socket_timeout=2
        )

    redis_client.ping()
    logger.info(f"Connected to Redis successfully at {REDIS_URL}")
except Exception as e:
    logger.warning(f"Could not connect to Redis ({e}). Running in fallback mode.")
    redis_client = None

# In-memory storage fallback when Redis is unreachable
memory_jobs = {}
memory_queue = []
