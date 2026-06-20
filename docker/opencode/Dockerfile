FROM node:22-slim
RUN apt-get update && apt-get install -y --no-install-recommends python3 && rm -rf /var/lib/apt/lists/*
RUN npm install -g opencode-ai
RUN useradd -m -u 1001 loop && \
    mkdir -p /home/loop/.local/share/opencode /home/loop/.local/state && \
    chown -R loop:loop /home/loop/.local
USER loop
COPY --chown=loop:loop http_loop_agent.py /app/http_loop_agent.py
COPY --chown=loop:loop claude_stream.py /app/claude_stream.py
COPY --chown=loop:loop opencode_stream.py /app/opencode_stream.py
COPY --chown=loop:loop opencode/opencode_http.py /app/opencode_http.py
EXPOSE 8090
CMD ["python3", "/app/opencode_http.py"]
