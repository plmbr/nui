/**
 * nui programmatic extension SDK — override-based base class.
 */

import * as readline from "node:readline";

export type RunContext = Record<string, unknown>;
export type MentionContext = Record<string, unknown>;

export class NuiExtension {
  apiVersion = "nui.plmbr.dev/extension/v1";
  extensionDir = process.env.NUI_EXTENSION_DIR ?? "";
  extensionName = process.env.NUI_EXTENSION_NAME ?? "";
  apiUrl = process.env.NUI_API_URL ?? "http://127.0.0.1:8080";

  initialize(): void {}

  shutdown(): void {}

  getHarnesses(): Array<Record<string, unknown>> { return []; }
  getAgents(): Array<Record<string, unknown>> { return []; }
  getMCPServers(): Array<Record<string, unknown>> { return []; }
  getCustomMCPServers(): Array<Record<string, unknown>> { return []; }
  getSkills(): Array<Record<string, unknown>> { return []; }
  getCustomSkills(): Array<Record<string, unknown>> { return []; }
  getRules(): Array<Record<string, unknown>> { return []; }
  getMentionProviders(): Array<Record<string, unknown>> { return []; }
  getHITLChannels(): Array<Record<string, unknown>> { return []; }
  getDeployers(): Array<Record<string, unknown>> { return []; }

  async *runHarness(_harnessId: string, _message: string, _ctx: RunContext = {}): AsyncIterable<string | Record<string, unknown>> {}

  cancelHarness(_runId: string): void {}

  async listMentions(_providerId: string, _parent = "", _query = "", _limit = 20, _ctx: MentionContext = {}): Promise<Record<string, unknown>> {
    return { items: [], breadcrumb: [] };
  }

  async resolveMention(_providerId: string, _value: string, _ctx: MentionContext = {}): Promise<Record<string, unknown>> {
    return { text: _value };
  }

  async deliverHITL(_channelId: string, _request: Record<string, unknown>, _ctx: MentionContext = {}): Promise<Record<string, unknown>> {
    return { ok: true };
  }

  async deploy(_deployerId: string, _req: Record<string, unknown>, _ctx: RunContext = {}): Promise<Record<string, unknown>> {
    return { ok: false, error: "deploy not implemented" };
  }

  readBundled(path: string): string {
    const fs = require("node:fs") as typeof import("node:fs");
    const nodePath = require("node:path") as typeof import("node:path");
    const full = nodePath.isAbsolute(path) ? path : nodePath.join(this.extensionDir || process.cwd(), path);
    return fs.readFileSync(full, "utf8");
  }

  serve(): void {
    const rl = readline.createInterface({ input: process.stdin });
    rl.on("line", (line) => {
      if (!line.trim()) return;
      let req: { id?: number | string; method?: string; params?: Record<string, unknown> };
      try {
        req = JSON.parse(line);
      } catch {
        return;
      }
      void this.dispatch(req);
    });
  }

  private write(msg: object): void {
    process.stdout.write(JSON.stringify(msg) + "\n");
  }

  private manifest(): Record<string, unknown> {
    return {
      apiVersion: this.apiVersion,
      name: this.extensionName,
      harnesses: this.getHarnesses(),
      agents: this.getAgents(),
      mcpServers: this.getMCPServers(),
      customMCPServers: this.getCustomMCPServers(),
      skills: this.getSkills(),
      customSkills: this.getCustomSkills(),
      rules: this.getRules(),
      mentionProviders: this.getMentionProviders(),
      hitlChannels: this.getHITLChannels(),
      agentDeployers: this.getDeployers(),
    };
  }

  private async dispatch(req: { id?: number | string; method?: string; params?: Record<string, unknown> }): Promise<void> {
    const { id: rid, method = "", params = {} } = req;

    if (method === "extension.initialize") {
      this.initialize();
      this.write({ jsonrpc: "2.0", id: rid, result: this.manifest() });
      return;
    }
    if (method === "extension.shutdown") {
      this.shutdown();
      this.write({ jsonrpc: "2.0", id: rid, result: { ok: true } });
      process.exit(0);
    }
    if (method === "harness.run") {
      const runId = String(params.runId ?? crypto.randomUUID());
      const harnessId = String(params.harnessId ?? "");
      const message = String(params.message ?? "");
      try {
        for await (const chunk of this.runHarness(harnessId, message, params)) {
          const event = typeof chunk === "string" ? { type: "text", content: chunk } : chunk;
          this.write({ jsonrpc: "2.0", method: "harness.event", params: { runId, ...event } });
        }
      } catch (e) {
        this.write({ jsonrpc: "2.0", method: "harness.event", params: { runId, type: "error", error: String(e) } });
      }
      this.write({ jsonrpc: "2.0", method: "harness.event", params: { runId, type: "done" } });
      this.write({ jsonrpc: "2.0", id: rid, result: { runId } });
      return;
    }
    if (method === "harness.cancel") {
      this.cancelHarness(String(params.runId ?? ""));
      this.write({ jsonrpc: "2.0", id: rid, result: { ok: true } });
      return;
    }
    if (method === "mention.list") {
      const result = await this.listMentions(String(params.providerId ?? ""), String(params.parent ?? ""), String(params.query ?? ""), Number(params.limit ?? 20), params);
      this.write({ jsonrpc: "2.0", id: rid, result });
      return;
    }
    if (method === "mention.resolve") {
      const result = await this.resolveMention(String(params.providerId ?? ""), String(params.value ?? ""), params);
      this.write({ jsonrpc: "2.0", id: rid, result });
      return;
    }
    if (method === "hitl.deliver") {
      const result = await this.deliverHITL(String(params.channelId ?? ""), (params.request as Record<string, unknown>) ?? {}, params);
      this.write({ jsonrpc: "2.0", id: rid, result });
      return;
    }
    if (method === "extension.deploy") {
      const result = await this.deploy(String(params.deployerId ?? ""), params, params);
      this.write({ jsonrpc: "2.0", id: rid, result });
      return;
    }
    this.write({ jsonrpc: "2.0", id: rid, error: { code: -32601, message: `method not found: ${method}` } });
  }
}
