#!/usr/bin/env python3
"""
Docker agent deployer for nui extensions.

Reads one JSON deploy request from stdin, writes one JSON response to stdout.
Registry, push, and run behavior come from deploy-config.yaml or env vars.
"""

from __future__ import annotations

import json
import os
import re
import shutil
import subprocess
import sys
import tempfile
from pathlib import Path
from typing import Any


def emit(resp: dict[str, Any]) -> None:
    sys.stdout.write(json.dumps(resp) + "\n")
    sys.stdout.flush()


def fail(msg: str) -> None:
    emit({"ok": False, "error": msg})
    sys.exit(1)


def load_config(ext_dir: Path) -> dict[str, Any]:
    cfg: dict[str, Any] = {
        "registry": "",
        "baseImage": "python:3.12-slim",
        "push": False,
        "run": False,
        "containerPort": 9090,
    }
    cfg_path = ext_dir / "deploy-config.yaml"
    if cfg_path.is_file():
        try:
            import yaml
        except ImportError:
            yaml = None
        if yaml is not None:
            with cfg_path.open() as f:
                loaded = yaml.safe_load(f) or {}
            if isinstance(loaded, dict):
                cfg.update(loaded)
    if os.environ.get("DOCKER_REGISTRY", "").strip():
        cfg["registry"] = os.environ["DOCKER_REGISTRY"].strip()
    if os.environ.get("DOCKER_DEPLOY_PUSH", "").lower() in ("1", "true", "yes"):
        cfg["push"] = True
    if os.environ.get("DOCKER_DEPLOY_RUN", "").lower() in ("1", "true", "yes"):
        cfg["run"] = True
    if os.environ.get("DOCKER_BASE_IMAGE", "").strip():
        cfg["baseImage"] = os.environ["DOCKER_BASE_IMAGE"].strip()
    return cfg


def sanitize_tag(value: str) -> str:
    value = re.sub(r"[^a-zA-Z0-9._-]+", "-", value.strip()).strip("-")
    return value or "latest"


def cfg_port(cfg: dict[str, Any], definition: dict[str, Any]) -> int:
    harness = definition.get("harness") or {}
    return int(harness.get("containerPort") or cfg.get("containerPort") or 9090)


def docker_available() -> bool:
    try:
        subprocess.run(["docker", "version"], capture_output=True, check=True)
        return True
    except (OSError, subprocess.CalledProcessError):
        return False


def run_cmd(cmd: list[str], cwd: Path | None = None) -> None:
    proc = subprocess.run(cmd, cwd=cwd, capture_output=True, text=True)
    if proc.returncode != 0:
        msg = proc.stderr.strip() or proc.stdout.strip() or "command failed"
        raise RuntimeError(msg)


def deploy(req: dict[str, Any], ext_dir: Path) -> dict[str, Any]:
    action = req.get("action")
    if action != "deploy":
        fail(f"unsupported action {action!r}")

    cfg = load_config(ext_dir)
    definition = req.get("definition") or {}
    agent_id = sanitize_tag(req.get("agentId") or definition.get("id") or "agent")
    version = sanitize_tag(definition.get("version") or "latest")
    local_tag = f"nui-{agent_id}:{version}"
    port = cfg_port(cfg, definition)

    build_dir = Path(tempfile.mkdtemp(prefix="nui-deploy-"))
    try:
        shutil.copy(ext_dir / "nui_agent.py", build_dir / "nui_agent.py")
        (build_dir / "agent.yaml").write_text(json.dumps(definition, indent=2), encoding="utf-8")

        assets = req.get("assets") or {}
        skills_dir = build_dir / "skills"
        skills_dir.mkdir(exist_ok=True)
        for i, skill in enumerate(assets.get("skills") or []):
            name = sanitize_tag(skill.get("name") or f"skill-{i}")
            skill_dir = skills_dir / name
            skill_dir.mkdir(parents=True, exist_ok=True)
            path = skill.get("path")
            content = skill.get("content")
            if path and Path(path).is_dir():
                shutil.copytree(path, skill_dir, dirs_exist_ok=True)
            elif content:
                (skill_dir / "SKILL.md").write_text(content, encoding="utf-8")

        rules_dir = build_dir / "rules"
        rules_dir.mkdir(exist_ok=True)
        for i, rule in enumerate(assets.get("rules") or []):
            name = sanitize_tag(rule.get("name") or f"rule-{i}")
            (rules_dir / f"{name}.md").write_text(rule.get("content") or "", encoding="utf-8")

        runner = f'''#!/usr/bin/env python3
import json
from pathlib import Path
from nui_agent import NuiAgent

class DeployedAgent(NuiAgent):
    def __init__(self):
        data = json.loads(Path("/app/agent.yaml").read_text())
        self.name = data.get("id") or data.get("name") or "deployed-agent"
        self.version = data.get("version") or "0.1.0"
        self._system_prompt = data.get("systemPrompt") or ""

    def run(self, message: str, **kwargs):
        system_prompt = kwargs.get("systemPrompt") or self._system_prompt
        if system_prompt:
            yield f"[system]\\n{{system_prompt}}\\n\\n"
        yield f"[user]\\n{{message}}\\n"

if __name__ == "__main__":
    DeployedAgent().serve()
'''
        (build_dir / "agent_runner.py").write_text(runner, encoding="utf-8")

        dockerfile = f"""FROM {cfg['baseImage']}
WORKDIR /app
COPY nui_agent.py agent_runner.py agent.yaml ./
COPY skills/ ./skills/
COPY rules/ ./rules/
EXPOSE {port}
CMD ["python3", "agent_runner.py", "--port", "{port}"]
"""
        (build_dir / "Dockerfile").write_text(dockerfile, encoding="utf-8")

        dry_run = os.environ.get("NUI_DEPLOY_DRY_RUN", "").lower() in ("1", "true", "yes")
        image_ref = local_tag
        registry = str(cfg.get("registry") or "").strip().rstrip("/")
        if registry:
            image_ref = f"{registry}/nui-{agent_id}:{version}"

        if not dry_run:
            if not docker_available():
                fail("docker is not available")
            run_cmd(["docker", "build", "-t", local_tag, "."], cwd=build_dir)
            if registry:
                run_cmd(["docker", "tag", local_tag, image_ref])
            if cfg.get("push") and registry:
                run_cmd(["docker", "push", image_ref])

        endpoint = None
        message = f"built image {image_ref}"
        if cfg.get("run") and not dry_run:
            run_cmd([
                "docker", "run", "-d", "--rm",
                "-p", f"127.0.0.1:0:{port}",
                local_tag,
            ], cwd=build_dir)
            message += " and started container"
            endpoint = {"host": "127.0.0.1", "port": port}

        return {
            "ok": True,
            "deploymentId": f"nui-{agent_id}-{version}",
            "status": "ready",
            "message": message,
            "endpoint": endpoint,
        }
    finally:
        shutil.rmtree(build_dir, ignore_errors=True)


def main() -> None:
    ext_dir = Path(os.environ.get("NUI_EXTENSION_DIR", Path(__file__).resolve().parent))
    line = sys.stdin.readline()
    if not line.strip():
        fail("empty request")
    try:
        req = json.loads(line)
    except json.JSONDecodeError as exc:
        fail(f"invalid json: {exc}")

    try:
        resp = deploy(req, ext_dir)
    except Exception as exc:
        fail(str(exc))
    emit(resp)


if __name__ == "__main__":
    main()
