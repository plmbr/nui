FROM node:22-slim

RUN apt-get update && apt-get install -y --no-install-recommends python3 \
    && rm -rf /var/lib/apt/lists/*

RUN npm install -g @earendil-works/pi-coding-agent

RUN useradd -m -u 1001 loop
USER loop
RUN mkdir -p /home/loop/.pi/agent/sessions

COPY --chown=loop:loop http_loop_agent.py /app/http_loop_agent.py
COPY --chown=loop:loop pi/pi_http.py /app/pi_http.py

EXPOSE 8090

CMD ["python3", "/app/pi_http.py"]
