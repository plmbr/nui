import { NuiAgent } from './nui_agent'

class EchoAgent extends NuiAgent {
  name = 'echo-agent'
  version = '0.1.0'

  async *run(message: string, _runId: string) {
    yield `Echo: ${message}\n`
  }
}

new EchoAgent().serve()
