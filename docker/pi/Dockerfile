FROM node:22-slim

RUN apt-get update && apt-get install -y --no-install-recommends python3 \
    && rm -rf /var/lib/apt/lists/*

RUN npm install -g @earendil-works/pi-coding-agent

RUN useradd -m -u 1001 nui
USER nui
RUN mkdir -p /home/nui/.pi/agent/sessions

COPY --chown=nui:nui http_nui_agent.py /app/http_nui_agent.py
COPY --chown=nui:nui harness_env.py /app/harness_env.py
COPY --chown=nui:nui claude_stream.py /app/claude_stream.py
COPY --chown=nui:nui pi_stream.py /app/pi_stream.py
COPY --chown=nui:nui pi_session.py /app/pi_session.py
COPY --chown=nui:nui pi/pi_http.py /app/pi_http.py

EXPOSE 8090

CMD ["python3", "/app/pi_http.py"]
