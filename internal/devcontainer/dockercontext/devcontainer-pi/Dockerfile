FROM node:22-slim

RUN apt-get update && apt-get install -y --no-install-recommends git ca-certificates \
    && rm -rf /var/lib/apt/lists/*

RUN npm install -g @earendil-works/pi-coding-agent

RUN useradd -m -u 1001 loop
USER loop
RUN mkdir -p /home/loop/.pi/agent/sessions
WORKDIR /workspaces/loop

CMD ["sleep", "infinity"]
