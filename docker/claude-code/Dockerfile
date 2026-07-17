FROM node:22-slim

RUN apt-get update && apt-get install -y --no-install-recommends python3 \
    && rm -rf /var/lib/apt/lists/*

RUN npm install -g @anthropic-ai/claude-code

RUN useradd -m -u 1001 nui
USER nui

COPY --chown=nui:nui http_nui_agent.py /app/http_nui_agent.py
COPY --chown=nui:nui harness_env.py /app/harness_env.py
COPY --chown=nui:nui claude_stream.py /app/claude_stream.py
COPY --chown=nui:nui claude_session.py /app/claude_session.py
COPY --chown=nui:nui claude-code/claude_code_http.py /app/claude_code_http.py

EXPOSE 8090

CMD ["python3", "/app/claude_code_http.py"]
