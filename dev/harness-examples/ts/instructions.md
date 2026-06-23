# TCP JSON-RPC Harness Examples — TypeScript (Reference)

These examples demonstrate the TCP JSON-RPC 2.0 protocol for custom Loop harnesses. **They are not wired to `Manager.GetAgent()` today.** See [harness-design.md](../../harness-design.md) for the production architecture.

## Files

- `loop_agent.ts` — harness framework base class
- `echo_agent.ts` — minimal echo harness
- `client.ts` — sample TCP client

## Setup

```sh
cd dev/harness-examples/ts
npm install
```

## Running

```sh
# terminal 1
npx ts-node echo_agent.ts

# terminal 2
npx ts-node client.ts
```

## Framework

Subclass `LoopAgent` and override `run()`:

```typescript
import { LoopAgent } from './loop_agent'

class MyAgent extends LoopAgent {
  name = 'my-agent'
  version = '0.1.0'

  async *run(message: string, _runId: string) {
    yield 'Thinking...'
    yield `\n\nYou said: ${message}\n`
  }
}

new MyAgent().serve()
```

### Hooks

- `onStart(port)` — after server binds and connection file is written
- `onCancel(runId)` — on `harness.cancel`
- `onShutdown()` — on `harness.shutdown` or signal

### Connection file

`serve()` writes `~/.loop/connections/<name>.json`:

```json
{"host": "127.0.0.1", "port": 52341, "session_id": "...", "pid": 9876}
```

### Methods

| Method | Description |
|---|---|
| `harness.info` | Metadata |
| `harness.run` | Stream `harness.event` notifications |
| `harness.cancel` | Cancel run |
| `harness.shutdown` | Release resources |

## Dependencies

`typescript`, `ts-node`, `@types/node` — devDependencies only.
