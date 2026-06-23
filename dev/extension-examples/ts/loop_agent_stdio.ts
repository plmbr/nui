/**
 * Loop harness framework — stdio JSON-RPC for ~/.loop/extensions/.
 *
 * Usage:
 *   import { LoopAgent } from "./loop_agent_stdio.js";
 *   class Echo extends LoopAgent {
 *     name = "echo";
 *     run(message) { return [message]; }
 *   }
 *   new Echo().serveStdio();
 */

import * as readline from "node:readline";

export class LoopAgent {
  name = "loop-agent";
  version = "0.1.0";

  run(_message: string, _runId: string, _kwargs: Record<string, string> = {}): string[] {
    throw new Error("override run()");
  }

  serveStdio(): void {
    const harnessId = process.env.LOOP_HARNESS_ID ?? this.name;
    const rl = readline.createInterface({ input: process.stdin });
    rl.on("line", (line) => {
      if (!line.trim()) return;
      let req: { id?: number; method?: string; params?: Record<string, string> };
      try {
        req = JSON.parse(line);
      } catch {
        return;
      }
      this.dispatch(req, harnessId);
    });
  }

  private write(msg: object): void {
    process.stdout.write(JSON.stringify(msg) + "\n");
  }

  private dispatch(req: { id?: number; method?: string; params?: Record<string, string> }, harnessId: string): void {
    const { id: rid, method = "", params = {} } = req;
    if (method === "harness.info") {
      this.write({ jsonrpc: "2.0", id: rid, result: { id: harnessId, name: this.name, version: this.version } });
      return;
    }
    if (method === "harness.run") {
      const runId = params.runId ?? crypto.randomUUID();
      for (const chunk of this.run(params.message ?? "", runId, params)) {
        this.write({ jsonrpc: "2.0", method: "harness.event", params: { runId, type: "text", content: chunk } });
      }
      this.write({ jsonrpc: "2.0", method: "harness.event", params: { runId, type: "done" } });
      this.write({ jsonrpc: "2.0", id: rid, result: { runId } });
      return;
    }
    if (method === "harness.shutdown") {
      this.write({ jsonrpc: "2.0", id: rid, result: { ok: true } });
      process.exit(0);
    }
    this.write({ jsonrpc: "2.0", id: rid, error: { code: -32601, message: `method not found: ${method}` } });
  }
}
