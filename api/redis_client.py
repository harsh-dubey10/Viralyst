# redis_client.py
# Shared Redis client configuration for Viralyst FastAPI backend.
#
# Redis is required: it is the only channel between this API and the Go worker,
# so jobs cannot be processed without it. The client is created lazily-connecting
# (redis-py reconnects on demand), so the API recovers automatically if Redis
# starts after the API does.

import os
import redis
import logging

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger("viralyst.redis")

REDIS_URL = os.getenv("REDIS_URL", "redis://localhost:6379/0")
JOB_TTL_SECONDS = 3600

if not (REDIS_URL.startswith("redis://") or REDIS_URL.startswith("rediss://")):
    REDIS_URL = f"redis://{REDIS_URL}"

redis_client = redis.Redis.from_url(
    REDIS_URL,
    decode_responses=True,
    socket_connect_timeout=2,
    socket_timeout=2,
    health_check_interval=30,
)


def redis_available() -> bool:
    try:
        return bool(redis_client.ping())
    except redis.RedisError:
        return False


if redis_available():
    logger.info(f"Connected to Redis at {REDIS_URL}")
else:
    logger.warning(f"Redis not reachable at {REDIS_URL} — /api/simulate will return 503 until it is.")
