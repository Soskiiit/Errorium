FROM python:3.13-slim

WORKDIR /app

COPY services/search_engine_core/requirements.txt /app/requirements.txt
RUN pip install --no-cache-dir --no-deps -r requirements.txt typing-extensions

COPY services/search_engine_core/src /app/src
COPY services/search_engine_indexer/src/indexer /app/indexer_src/indexer

ENV PYTHONPATH=/app/src:/app/indexer_src
EXPOSE 50051

CMD ["python", "-m", "core.server"]
