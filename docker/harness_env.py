"""Helpers for applying nui session harness config in Docker agents."""

from __future__ import annotations

import os
from typing import Any


def subprocess_env(kwargs: dict[str, Any]) -> dict[str, str]:
    """Return a child-process env with optional ADL overrides from the HTTP /run body."""
    env = os.environ.copy()
    extra = kwargs.get("env")
    if isinstance(extra, dict):
        for key, value in extra.items():
            if value is not None:
                env[str(key)] = str(value)
    return env
