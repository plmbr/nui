# Loop Extension Examples — TypeScript

## Files

- `loop_agent.ts` — extension framework base class
- `echo_agent.ts` — minimal extension using the framework
- `client.ts` — sample client that connects to a running extension

## Setup

```sh
cd dev/extension-examples/ts
npm install
```

## Running

```sh
# terminal 1 — start the extension
npx ts-node echo_agent.ts

# terminal 2 — run the client
npx ts-node client.ts

# or specify a different extension name
npx ts-node client.ts my-agent
```

## How it works

### loop_agent.ts

Base class that handles all infrastructure. Subclass it and override `run()`:

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

`run()` is an async generator — you can `await` inside it before yielding chunks, which makes it natural for calling LLM APIs or doing async I/O.

Optional hooks:
- `onStart(port)` — called after the server binds and connection file is written
- `onCancel(runId)` — called when Loop sends `harness.cancel`

On startup, `serve()` binds a random TCP port and writes `~/.loop/extensions/<name>.json`:

```json
{ "host": "127.0.0.1", "port": 52341, "session_id": "...", "pid": 9876 }
```

Loop reads this file to connect (or reconnect after a restart).

### client.ts

1. Reads `~/.loop/extensions/echo-agent.json` (exits if the extension is not running)
2. Connects over TCP
3. Calls `harness.info` and prints the result
4. Calls `harness.run` with a message — `harness.event` notifications are printed as they arrive (streaming), then prints the final result

## Dependencies

`typescript`, `ts-node`, `@types/node` — devDependencies only, no runtime deps.
