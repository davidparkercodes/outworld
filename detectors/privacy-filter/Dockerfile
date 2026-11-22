FROM python:3.12-slim

ENV PYTHONDONTWRITEBYTECODE=1 PYTHONUNBUFFERED=1 HF_HOME=/models ORT_DISABLE_TELEMETRY=1 HF_HUB_DISABLE_TELEMETRY=1
WORKDIR /app
ARG BACKEND=onnx
COPY requirements.txt requirements-torch.txt ./
RUN pip install --no-cache-dir -r requirements.txt
RUN if [ "$BACKEND" = "torch" ]; then pip install --no-cache-dir torch==2.15.0 --index-url https://download.pytorch.org/whl/cpu && pip install --no-cache-dir -r requirements-torch.txt; fi
ARG PRELOAD=1
ARG MODEL=openai/privacy-filter
ARG ONNX_FILE=onnx/model.onnx
ENV PRIVACY_FILTER_BACKEND=$BACKEND PRIVACY_FILTER_MODEL=$MODEL PRIVACY_FILTER_ONNX_FILE=$ONNX_FILE
RUN mkdir -p /models && if [ "$PRELOAD" = "1" ]; then if [ "$BACKEND" = "torch" ]; then WEIGHTS="model.safetensors"; else WEIGHTS="$ONNX_FILE*"; fi; python -c "import sys; from huggingface_hub import snapshot_download; snapshot_download(sys.argv[1], allow_patterns=['config.json', 'tokenizer.json', 'viterbi_calibration.json', sys.argv[2]])" "$MODEL" "$WEIGHTS"; fi
COPY app.py chunking.py decoding.py ./
RUN useradd --system --uid 10001 detector && chown -R detector /models /app
USER detector
EXPOSE 8791
HEALTHCHECK --interval=30s --timeout=5s --start-period=120s CMD python -c "import urllib.request; urllib.request.urlopen('http://127.0.0.1:8791/health')"
CMD ["uvicorn", "app:app", "--host", "0.0.0.0", "--port", "8791"]
