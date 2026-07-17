/**
 * nui extension catalog provider — stdio JSON-RPC.
 */

import * as readline from "node:readline";

export class NuiCatalog {
  serve(): void {
    const rl = readline.createInterface({ input: process.stdin });
    rl.on("line", (line) => {
      if (!line.trim()) return;
      let req: { id?: number; method?: string };
      try {
        req = JSON.parse(line);
      } catch {
        return;
      }
      this.dispatch(req);
    });
  }

  listHarnesses(): object[] {
    return [];
  }
  listMCPServers(): object[] {
    return [];
  }
  listSkills(): object[] {
    return [];
  }
  listAgents(): object[] {
    return [];
  }

  private write(msg: object): void {
    process.stdout.write(JSON.stringify(msg) + "\n");
  }

  private dispatch(req: { id?: number; method?: string }): void {
    const { id: rid, method = "" } = req;
    if (method === "extension.initialize") {
      this.write({
        jsonrpc: "2.0",
        id: rid,
        result: { apiVersion: "nui.dev/extension/v1", extensionName: process.env.NUI_EXTENSION_NAME ?? "" },
      });
      return;
    }
    if (method === "extension.listHarnesses") {
      this.write({ jsonrpc: "2.0", id: rid, result: { harnesses: this.listHarnesses() } });
      return;
    }
    if (method === "extension.listMCPServers") {
      this.write({ jsonrpc: "2.0", id: rid, result: { mcpServers: this.listMCPServers() } });
      return;
    }
    if (method === "extension.listSkills") {
      this.write({ jsonrpc: "2.0", id: rid, result: { skills: this.listSkills() } });
      return;
    }
    if (method === "extension.listAgents") {
      this.write({ jsonrpc: "2.0", id: rid, result: { agents: this.listAgents() } });
      return;
    }
    if (method === "extension.shutdown") {
      this.write({ jsonrpc: "2.0", id: rid, result: { ok: true } });
      process.exit(0);
    }
    this.write({ jsonrpc: "2.0", id: rid, error: { code: -32601, message: `method not found: ${method}` } });
  }
}
