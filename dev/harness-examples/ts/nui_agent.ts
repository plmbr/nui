/**
 * nui extension framework — TypeScript.
 *
 * Subclass NuiAgent, override `run()`, call `serve()`. Everything else
 * (TCP socket, connection file, JSON-RPC dispatch, streaming) is handled here.
 *
 * Example:
 *
 *   import { NuiAgent } from './nui_agent'
 *
 *   class MyAgent extends NuiAgent {
 *     name = 'my-agent'
 *     version = '0.1.0'
 *
 *     async *run(message: string, _runId: string) {
 *       yield 'Thinking...'
 *       yield `\n\nYou said: ${message}\n`
 *     }
 *   }
 *
 *   new MyAgent().serve()
 */

import * as net from 'net'
import * as fs from 'fs'
import * as os from 'os'
import * as path from 'path'
import { randomUUID } from 'crypto'

interface JsonRpcRequest {
  jsonrpc: '2.0'
  id?: number | string | null
  method: string
  params?: Record<string, unknown>
}

export abstract class NuiAgent {
  abstract name: string
  abstract version: string

  // ── Override this ─────────────────────────────────────────────────────────

  abstract run(message: string, runId: string): AsyncGenerator<string>

  // ── Optional hooks ────────────────────────────────────────────────────────

  onStart(_port: number): void {}
  onCancel(_runId: string): void {}

  // ── Public ────────────────────────────────────────────────────────────────

  serve(): void {
    const sessionId = randomUUID()
    const server = net.createServer((socket) => this._handleClient(socket))

    server.listen(0, '127.0.0.1', () => {
      const addr = server.address() as net.AddressInfo
      const connFile = this._writeConnectionFile(addr.port, sessionId)
      this._log(`listening on 127.0.0.1:${addr.port}`)
      this._log(`connection file: ${connFile}`)
      this.onStart(addr.port)
    })

    const cleanup = () => {
      this._log('shutting down')
      this._removeConnectionFile()
      server.close()
      process.exit(0)
    }
    process.on('SIGINT', cleanup)
    process.on('SIGTERM', cleanup)
  }

  // ── Private ───────────────────────────────────────────────────────────────

  private _handleClient(socket: net.Socket): void {
    const addr = `${socket.remoteAddress}:${socket.remotePort}`
    this._log(`client connected: ${addr}`)
    let buf = ''

    socket.on('data', (chunk) => {
      buf += chunk.toString()
      let nl: number
      while ((nl = buf.indexOf('\n')) !== -1) {
        const line = buf.slice(0, nl).trim()
        buf = buf.slice(nl + 1)
        if (!line) continue
        try {
          this._dispatch(socket, JSON.parse(line) as JsonRpcRequest)
        } catch (e) {
          this._log(`dispatch error: ${e}`)
        }
      }
    })

    socket.on('close', () => this._log(`client disconnected: ${addr}`))
    socket.on('error', (e) => this._log(`socket error: ${e.message}`))
  }

  private _dispatch(socket: net.Socket, req: JsonRpcRequest): void {
    const { method, params = {}, id } = req

    if (method === 'harness.info') {
      this._send(socket, {
        jsonrpc: '2.0', id,
        result: { name: this.name, version: this.version, capabilities: ['run', 'cancel'] },
      })
      return
    }

    if (method === 'harness.run') {
      const message = (params.message as string) ?? ''
      const runId = (params.runId as string) ?? randomUUID()
      void this._streamRun(socket, id, message, runId)
      return
    }

    if (method === 'harness.cancel') {
      this.onCancel((params.runId as string) ?? '')
      this._send(socket, { jsonrpc: '2.0', id, result: { ok: true } })
      return
    }

    this._send(socket, {
      jsonrpc: '2.0', id,
      error: { code: -32601, message: `method not found: ${method}` },
    })
  }

  private async _streamRun(
    socket: net.Socket,
    id: JsonRpcRequest['id'],
    message: string,
    runId: string,
  ): Promise<void> {
    try {
      for await (const chunk of this.run(message, runId)) {
        this._send(socket, {
          jsonrpc: '2.0', method: 'harness.event',
          params: { runId, type: 'text', content: chunk },
        })
      }
    } catch (e) {
      this._send(socket, {
        jsonrpc: '2.0', method: 'harness.event',
        params: { runId, type: 'error', error: String(e) },
      })
    }
    this._send(socket, {
      jsonrpc: '2.0', method: 'harness.event',
      params: { runId, type: 'done' },
    })
    this._send(socket, { jsonrpc: '2.0', id, result: { runId } })
  }

  private _send(socket: net.Socket, msg: object): void {
    socket.write(JSON.stringify(msg) + '\n')
  }

  private _connectionId(): string {
    return process.env.NUI_CONNECTION_ID ?? this.name
  }

  private _connectionFilePath(): string {
    return path.join(os.homedir(), '.nui', 'connections', `${this._connectionId()}.json`)
  }

  private _writeConnectionFile(port: number, sessionId: string): string {
    const filePath = this._connectionFilePath()
    fs.mkdirSync(path.dirname(filePath), { recursive: true })
    fs.writeFileSync(filePath, JSON.stringify({
      host: '127.0.0.1',
      port,
      session_id: sessionId,
      pid: process.pid,
    }))
    return filePath
  }

  private _removeConnectionFile(): void {
    try { fs.unlinkSync(this._connectionFilePath()) } catch { /* already gone */ }
  }

  private _log(msg: string): void {
    process.stderr.write(`[${this.name}] ${msg}\n`)
  }
}
