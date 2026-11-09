FROM python:3.12-slim

ENV PYTHONDONTWRITEBYTECODE=1 PYTHONUNBUFFERED=1 HF_HOME=/models
WORKDIR /app
RUN pip install --no-cache-dir torch --index-url https://download.pytorch.org/whl/cpu
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
ARG PRELOAD=1
RUN if [ "$PRELOAD" = "1" ]; then python -c "from gliner import GLiNER; GLiNER.from_pretrained('urchade/gliner_multi_pii-v1')"; fi
COPY app.py .
RUN useradd --system --uid 10001 gliner && chown -R gliner /models /app
USER gliner
EXPOSE 8790
HEALTHCHECK --interval=30s --timeout=5s --start-period=60s CMD python -c "import urllib.request; urllib.request.urlopen('http://127.0.0.1:8790/health')"
CMD ["uvicorn", "app:app", "--host", "0.0.0.0", "--port", "8790"]
