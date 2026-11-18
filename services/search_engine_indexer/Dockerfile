# Search Engine Indexer (Роль 1): офлайн-индексация дампа Stack Overflow.
# Только Python (stdlib) + 7-Zip для потокового чтения архива (`7zz x -so`).
FROM python:3.13-alpine

RUN apk add --no-cache 7zip

WORKDIR /app
COPY src/ /app/src/

ENV PYTHONPATH=/app/src \
    PYTHONUNBUFFERED=1

# python -m indexer sample | corpus  (см. docs/Tech/03-data.md, п. 4.10)
ENTRYPOINT ["python", "-m", "indexer"]
