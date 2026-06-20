# Loop UI

React + Vite frontend for the Loop agent chat interface.

## Development

```sh
npm install
npm run dev     # :5173, proxies /api → Go server on :8080
npm run build   # output to dist/ (embedded by Go binary)
npm run lint
```

Run the Go server separately (`go run . ui` from repo root). Vite proxies API calls so HMR works without rebuilding the Go binary.

## Key files

| Path | Role |
|---|---|
| `src/App.tsx` | Session list + layout |
| `src/hooks/useSessionChat.ts` | AG-UI client (`@ag-ui/client`); primary chat transport |
| `src/components/ChatPanel.tsx` | Message rendering, tool calls, images |
| `src/components/NewSessionDialog.tsx` | Builtin harness picker + custom ADL agents |
| `src/api.ts` | REST client |
| `src/types.ts` | TypeScript types mirroring Go models |

## Chat transport

The UI streams chat via `POST /api/sessions/:id/ag-ui` using the [AG-UI protocol](https://github.com/ag-ui-protocol/ag-ui), not the legacy `/chat` endpoint.

On session select, messages load from `GET /api/sessions/:id/messages` (persisted in `~/.loop/data.json`), falling back to `GET /api/sessions/:id/history` (agent session files).

## Stack

Tailwind CSS v4, shadcn/ui (Base UI), `react-markdown`, `@ag-ui/client`.

See [CLAUDE.md](../CLAUDE.md) for full project documentation.
