FROM node:22-slim
RUN apt-get update && apt-get install -y --no-install-recommends python3 && rm -rf /var/lib/apt/lists/*
RUN npm install -g opencode-ai
RUN useradd -m -u 1001 nui && \
    mkdir -p /home/nui/.local/share/opencode /home/nui/.local/state && \
    chown -R nui:nui /home/nui/.local
USER nui
COPY --chown=nui:nui http_nui_agent.py /app/http_nui_agent.py
COPY --chown=nui:nui harness_env.py /app/harness_env.py
COPY --chown=nui:nui claude_stream.py /app/claude_stream.py
COPY --chown=nui:nui opencode_stream.py /app/opencode_stream.py
COPY --chown=nui:nui opencode_session.py /app/opencode_session.py
COPY --chown=nui:nui opencode/opencode_http.py /app/opencode_http.py
EXPOSE 8090
CMD ["python3", "/app/opencode_http.py"]
