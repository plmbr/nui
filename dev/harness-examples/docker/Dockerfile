FROM python:3.12-slim

WORKDIR /app
COPY nui_agent.py echo_agent.py ./

EXPOSE 9090

CMD ["python3", "echo_agent.py", "--port", "9090"]
