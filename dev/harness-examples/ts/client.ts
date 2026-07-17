/**
 * Sample nui extension client — TypeScript.
 *
 * Reads ~/.nui/connections/<name>.json, connects over TCP, and calls
 * harness.info then harness.run, printing streamed events as they arrive.
 *
 * Usage:
 *   # terminal 1 — start the agent
 *   npx ts-node echo_agent.ts
 *
 *   # terminal 2 — run this client
 *   npx ts-node client.ts [extension-name]   # default: echo-agent
 */

import * as net from 'net'
import * as fs from 'fs'
import * as os from 'os'
import * as path from 'path'
import { randomUUID } from 'crypto'

interface ConnectionInfo {
  host: string
  port: number
  session_id: string
  pid: number
}

interface JsonRpcMessage {
  jsonrpc: '2.0'
  id?: number | string | null
  method?: string
  params?: Record<string, unknown>
  result?: unknown
  error?: { code: number; message: string }
}

function loadConnection(name: string): ConnectionInfo {
  const filePath = path.join(os.homedir(), '.nui', 'connections', `${name}.json`)
  if (!fs.existsSync(filePath)) {
    console.error(`connection file not found: ${filePath}\nIs the extension running?`)
    process.exit(1)
  }
  return JSON.parse(fs.readFileSync(filePath, 'utf8')) as ConnectionInfo
}

class Client {
  private socket: net.Socket
  private buf = ''
  private nextId = 1
  private pending = new Map<number, (msg: JsonRpcMessage) => void>()
  private onEvent?: (params: Record<string, unknown>) => void

  constructor(socket: net.Socket) {
    this.socket = socket
    socket.on('data', (chunk) => {
      this.buf += chunk.toString()
      let nl: number
      while ((nl = this.buf.indexOf('\n')) !== -1) {
        const line = this.buf.slice(0, nl).trim()
        this.buf = this.buf.slice(nl + 1)
        if (!line) continue
        const msg = JSON.parse(line) as JsonRpcMessage
        if (msg.id != null && this.pending.has(msg.id as number)) {
          this.pending.get(msg.id as number)!(msg)
          this.pending.delete(msg.id as number)
        } else if (msg.method === 'harness.event' && this.onEvent) {
          this.onEvent(msg.params ?? {})
        }
      }
    })
  }

  call(method: string, params: Record<string, unknown> = {}): Promise<unknown> {
    return new Promise((resolve, reject) => {
      const id = this.nextId++
      this.pending.set(id, (msg) => {
        if (msg.error) reject(new Error(msg.error.message))
        else resolve(msg.result)
      })
      this.socket.write(JSON.stringify({ jsonrpc: '2.0', id, method, params }) + '\n')
    })
  }

  async run(message: string, onEvent: (e: Record<string, unknown>) => void): Promise<unknown> {
    this.onEvent = onEvent
    const result = await this.call('harness.run', { message, runId: randomUUID() })
    this.onEvent = undefined
    return result
  }
}

async function main() {
  const name = process.argv[2] ?? 'echo-agent'
  const info = loadConnection(name)

  console.log(`connecting to extension: ${name}`)
  console.log(`  host=${info.host} port=${info.port} pid=${info.pid}`)

  const socket = net.connect(info.port, info.host)
  await new Promise<void>((resolve) => socket.once('connect', resolve))

  const client = new Client(socket)

  // 1. Get extension info
  const extInfo = await client.call('harness.info')
  console.log(`\nharness.info →`, JSON.stringify(extInfo, null, 2))

  // 2. Run with streaming events
  console.log('\nharness.run →')
  const result = await client.run('Hello from the nui client!', (event) => {
    if (event.type === 'text') process.stdout.write(event.content as string)
    if (event.type === 'done') process.stdout.write('\n')
  })
  console.log('result:', result)

  socket.destroy()
}

main().catch((e) => { console.error(e); process.exit(1) })
