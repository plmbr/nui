FROM node:22-slim

RUN apt-get update && apt-get install -y --no-install-recommends git ca-certificates \
    && rm -rf /var/lib/apt/lists/*

RUN npm install -g @anthropic-ai/claude-code

RUN useradd -m -u 1001 nui
USER nui
WORKDIR /workspaces/nui

CMD ["sleep", "infinity"]
