# Корпус benchmark

30 документов `D01`–`D30` для проверки поиска (BM25) и ответов RAG. Разметка вопросов — в `../questions.json`, описание набора — в `docs/Tech/05-verification.md`, раздел 3.

## Состав

- `documents.jsonl` — по одному JSON-объекту в строке: `doc_id` (1–30, метка `D{doc_id:02d}`), `title`, `text` (`Question: …` и `Answer: …`), `tags`, `so_id`, `url`.
- Документ — тред Stack Overflow: заголовок, вопрос и **принятый** ответ, очищенные от HTML (переводы строк и отступы в блоках кода сохранены).
- 10 тем по 3 документа: Python (`IndexError`, `ModuleNotFoundError`), Java (`NullPointerException`, `OutOfMemoryError`), C++ (линкер, vtable), C# (async/await), Docker, Git, MySQL, CORS. Документы `D01`–`D10` — по одному на тему, `D11`–`D20` — вторые, `D21`–`D30` — третьи (тема `k` = `D{k}`, `D{k+10}`, `D{k+20}`).

## Происхождение и лицензия

Документы выбраны из выборки Stack Overflow, построенной индексатором (`make index-sample`, `SAMPLE_DENOM=100`), по реальным принятым ответам. Исходный пост указан в полях `so_id` и `url`. Содержимое Stack Overflow распространяется по лицензии CC BY-SA (версия 2.5, 3.0 или 4.0 в зависимости от даты публикации) с обязательной атрибуцией авторам: ссылка на оригинал приведена в каждом документе.

## Запуск

```bash
make index-corpus   # индекс корпуса за секунды → data/index-benchmark/
```

Режим `corpus` индексатора описан в `docs/Tech/03-data.md`, п. 4.9.
