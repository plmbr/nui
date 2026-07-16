"""Loop programmatic extension base class."""

from __future__ import annotations

import json
import os
import sys
import uuid
from typing import Any, Generator, Iterable


class LoopExtension:
    """Base class for programmatic Loop extensions.

    Subclass and override discovery (get_*) and runtime (run_harness, list_mentions, ...) methods.
    """

    api_version = "loop.dev/extension/v1"

    def __init__(self) -> None:
        self.extension_dir = os.environ.get("LOOP_EXTENSION_DIR", "")
        self.extension_name = os.environ.get("LOOP_EXTENSION_NAME", "")
        self.api_url = os.environ.get("LOOP_API_URL", "http://127.0.0.1:8080")
        self._emit_event = None
        self._active_ctx: dict[str, Any] | None = None

    @classmethod
    def loop_session_id(cls, ctx: dict[str, Any] | None = None) -> str:
        from loop_hitl import resolve_loop_session_id

        return resolve_loop_session_id(ctx=ctx)

    @classmethod
    def loop_run_id(cls, ctx: dict[str, Any] | None = None) -> str:
        from loop_hitl import resolve_loop_run_id

        return resolve_loop_run_id(ctx=ctx)

    # --- lifecycle ---

    def initialize(self) -> None:
        """Called once when Loop sends extension.initialize."""

    def shutdown(self) -> None:
        """Called on extension.shutdown before process exit."""

    # --- discovery (override to contribute) ---

    def get_harnesses(self) -> list[dict[str, Any]]:
        return []

    def get_agents(self) -> list[dict[str, Any]]:
        return []

    def get_mcp_servers(self) -> list[dict[str, Any]]:
        return []

    def get_custom_mcp_servers(self) -> list[dict[str, Any]]:
        return []

    def get_skills(self) -> list[dict[str, Any]]:
        return []

    def get_custom_skills(self) -> list[dict[str, Any]]:
        return []

    def get_rules(self) -> list[dict[str, Any]]:
        return []

    def get_mention_providers(self) -> list[dict[str, Any]]:
        return []

    def get_hitl_channels(self) -> list[dict[str, Any]]:
        return []

    def get_deployers(self) -> list[dict[str, Any]]:
        return []

    # --- runtime dispatch ---

    def run_harness(
        self, harness_id: str, message: str, ctx: dict[str, Any] | None = None
    ) -> Generator[str | dict[str, Any], None, None]:
        _ = harness_id, message, ctx
        if False:
            yield ""

    def cancel_harness(self, run_id: str) -> None:
        _ = run_id

    def list_mentions(
        self,
        provider_id: str,
        parent: str = "",
        query: str = "",
        limit: int = 20,
        ctx: dict[str, Any] | None = None,
    ) -> dict[str, Any]:
        _ = provider_id, parent, query, limit, ctx
        return {"items": [], "breadcrumb": []}

    def resolve_mention(
        self, provider_id: str, value: str, ctx: dict[str, Any] | None = None
    ) -> dict[str, Any]:
        _ = provider_id, value, ctx
        return {"text": value}

    def deliver_hitl(
        self, channel_id: str, request: dict[str, Any], ctx: dict[str, Any] | None = None
    ) -> dict[str, Any]:
        _ = channel_id, request, ctx
        return {"ok": True}

    def deploy(
        self, deployer_id: str, req: dict[str, Any], ctx: dict[str, Any] | None = None
    ) -> dict[str, Any]:
        _ = deployer_id, req, ctx
        return {"ok": False, "error": "deploy not implemented"}

    # --- helpers ---

    def read_bundled(self, path: str) -> str:
        base = self.extension_dir or os.getcwd()
        full = path if os.path.isabs(path) else os.path.join(base, path)
        with open(full, encoding="utf-8") as f:
            return f.read()

    def serve(self) -> None:
        """Start the stdio JSON-RPC loop."""
        for line in sys.stdin:
            line = line.strip()
            if not line:
                continue
            try:
                req = json.loads(line)
            except json.JSONDecodeError:
                continue
            self._dispatch(req)

    def _write(self, msg: dict[str, Any]) -> None:
        sys.stdout.write(json.dumps(msg) + "\n")
        sys.stdout.flush()

    def _manifest(self) -> dict[str, Any]:
        return {
            "apiVersion": self.api_version,
            "name": self.extension_name,
            "harnesses": self.get_harnesses(),
            "agents": self.get_agents(),
            "mcpServers": self.get_mcp_servers(),
            "customMCPServers": self.get_custom_mcp_servers(),
            "skills": self.get_skills(),
            "customSkills": self.get_custom_skills(),
            "rules": self.get_rules(),
            "mentionProviders": self.get_mention_providers(),
            "hitlChannels": self.get_hitl_channels(),
            "agentDeployers": self.get_deployers(),
        }

    def _with_active_ctx(self, ctx: dict[str, Any] | None):
        class _Ctx:
            def __init__(self, ext: LoopExtension, params: dict[str, Any] | None) -> None:
                self.ext = ext
                self.params = params

            def __enter__(self):
                self.ext._active_ctx = self.params
                return self.ext

            def __exit__(self, *args: Any) -> None:
                self.ext._active_ctx = None

        return _Ctx(self, ctx)

    def _dispatch(self, req: dict[str, Any]) -> None:
        method = req.get("method", "")
        params = req.get("params") or {}
        rid = req.get("id")

        if method == "extension.initialize":
            self.initialize()
            self._write({"jsonrpc": "2.0", "id": rid, "result": self._manifest()})
            return

        if method == "extension.shutdown":
            self.shutdown()
            self._write({"jsonrpc": "2.0", "id": rid, "result": {"ok": True}})
            sys.exit(0)

        if method == "harness.run":
            self._handle_harness_run(rid, params)
            return

        if method == "harness.cancel":
            self.cancel_harness(params.get("runId", ""))
            self._write({"jsonrpc": "2.0", "id": rid, "result": {"ok": True}})
            return

        if method == "harness.shutdown":
            self.shutdown()
            self._write({"jsonrpc": "2.0", "id": rid, "result": {"ok": True}})
            return

        if method == "mention.list":
            with self._with_active_ctx(params):
                result = self.list_mentions(
                    params.get("providerId", ""),
                    parent=params.get("parent", ""),
                    query=params.get("query", ""),
                    limit=int(params.get("limit", 20) or 20),
                    ctx=params,
                )
            self._write({"jsonrpc": "2.0", "id": rid, "result": result})
            return

        if method == "mention.resolve":
            with self._with_active_ctx(params):
                result = self.resolve_mention(
                    params.get("providerId", ""),
                    params.get("value", ""),
                    ctx=params,
                )
            self._write({"jsonrpc": "2.0", "id": rid, "result": result})
            return

        if method == "hitl.deliver":
            with self._with_active_ctx(params):
                result = self.deliver_hitl(
                    params.get("channelId", ""),
                    params.get("request") or {},
                    ctx=params,
                )
            self._write({"jsonrpc": "2.0", "id": rid, "result": result})
            return

        if method == "extension.deploy":
            with self._with_active_ctx(params):
                result = self.deploy(params.get("deployerId", ""), params, ctx=params)
            self._write({"jsonrpc": "2.0", "id": rid, "result": result})
            return

        self._write({
            "jsonrpc": "2.0",
            "id": rid,
            "error": {"code": -32601, "message": f"method not found: {method}"},
        })

    def _handle_harness_run(self, rid: Any, params: dict[str, Any]) -> None:
        harness_id = params.get("harnessId", "")
        message = params.get("message", "")
        run_id = params.get("runId") or str(uuid.uuid4())

        def emit(event: dict[str, Any]) -> None:
            self._write({
                "jsonrpc": "2.0",
                "method": "harness.event",
                "params": {"runId": run_id, **event},
            })

        self._emit_event = emit
        try:
            with self._with_active_ctx(params):
                for chunk in self.run_harness(harness_id, message, params):
                    if isinstance(chunk, dict):
                        emit(chunk)
                    else:
                        emit({"type": "text", "content": chunk})
        except Exception as e:
            emit({"type": "error", "error": str(e)})
        finally:
            self._emit_event = None
        emit({"type": "done"})
        self._write({"jsonrpc": "2.0", "id": rid, "result": {"runId": run_id}})

    def ask_user(self, **kwargs: Any) -> dict[str, Any]:
        from loop_hitl import ask_user as hitl_ask_user

        ctx = kwargs.pop("ctx", None) or self._active_ctx
        kwargs.setdefault("session_id", self.loop_session_id(ctx))
        kwargs.setdefault("run_id", self.loop_run_id(ctx))
        return hitl_ask_user(
            emit_event=self._emit_event,
            ctx=ctx,
            **kwargs,
        )

    def request_approval(self, **kwargs: Any) -> dict[str, Any]:
        from loop_hitl import request_approval as hitl_request_approval

        ctx = kwargs.pop("ctx", None) or self._active_ctx
        kwargs.setdefault("session_id", self.loop_session_id(ctx))
        kwargs.setdefault("run_id", self.loop_run_id(ctx))
        return hitl_request_approval(
            emit_event=self._emit_event,
            ctx=ctx,
            **kwargs,
        )
