# Search Engine Core (Роль 2): BM25 с нуля, mmap-чтение индекса, gRPC :50051.
# Контекст сборки — корень репозитория (protos/ + общие модули индексатора).
FROM python:3.13-slim

WORKDIR /app

COPY services/search_engine_core/requirements.txt /app/requirements.txt
RUN pip install --no-cache-dir -r /app/requirements.txt

COPY services/search_engine_core/src/ /app/src/

# Общий токенизатор и формат индекса — единый источник истины (docs/Tech/03-data.md, п. 5.6)
COPY services/search_engine_indexer/src/indexer/__init__.py \
     services/search_engine_indexer/src/indexer/tokenizer.py \
     services/search_engine_indexer/src/indexer/index_format.py \
     /app/src/indexer/

# Кодогенерация pb2 из контрактов (в git не коммитятся)
COPY protos/ /app/protos/
RUN python -m grpc_tools.protoc -I/app/protos \
    --python_out=/app/src --grpc_python_out=/app/src \
    /app/protos/search.proto /app/protos/graph.proto

ENV PYTHONPATH=/app/src \
    PYTHONUNBUFFERED=1 \
    INDEX_DIR=/app/data/index \
    SEARCH_GRPC_PORT=50051

EXPOSE 50051

# python -m core  (см. docs/Tech/03-data.md, раздел 6)
ENTRYPOINT ["python", "-m", "core"]