# Human written features

> **Status:** Historical pre-implementation notes. Largely superseded by [`dev/dev.md`](../dev.md). Terminology here uses "projects" — the implementation uses "sessions".

1. nui UI allows creating and managing projects. Projects can be interactive chat sessions, autonomous agents, or semi-autonomus agents. Autonomous agents can be run at desired intervals. Semi-autonomus agents and interactive chat sessions usually require human in the loop. Human in the loop will be implemented as UI interaction on the chat interface for initial version. In future versions human in the loop will be done with other interfaces such as Slack, WhatsApp, Telegram etc.

2. This is a Go application that bundles the web UI. It will provide extensibility for various features.

3. Each project is assigned an agent type. Agent types are defined using ADL (Agent Definition Language).

4. ADL is a YAML format for defining agent types. In this definition, there will be system prompt, AI dependencies (Agent Skills, MCP Servers, Plugins, etc.) and steps for the agent to follow. Each agent step can have different AI dependencies and different system prompt, and different AI harness and model requirements.
[TODO]: design the ADL format.

5. nui will support Claude Code and Pi as AI harnesses. It will be extensible to support other AI harnesses and remote agent execution runtimes.
[TODO]: design the agent runtime extension interface. think about supporting Docker containers.

6. nui UI and backend are decoupled and will support reconnections. If the backend is restarted, the UI will reconnect to all existing projects ans show latest status.
