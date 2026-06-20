FROM node:22-slim

RUN apt-get update && apt-get install -y --no-install-recommends python3 ca-certificates \
    && rm -rf /var/lib/apt/lists/*

RUN npm install -g @openai/codex

RUN useradd -m -u 1001 loop
USER loop

COPY --chown=loop:loop http_loop_agent.py /app/http_loop_agent.py
COPY --chown=loop:loop claude_stream.py /app/claude_stream.py
COPY --chown=loop:loop codex_stream.py /app/codex_stream.py
COPY --chown=loop:loop codex/codex_http.py /app/codex_http.py

EXPOSE 8090

CMD ["python3", "/app/codex_http.py"]
