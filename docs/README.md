# Документация проекта Errorium

Данный раздел содержит полный комплект проектной и технической документации в соответствии со стандартом **Spec-Driven Development (SDD)** курса **НИРС (ОАД) МГТУ им. Н. Э. Баумана**.

---

## Структура документации

```text
docs/
├── README.md                      # Данная карта документов
├── document-manifest.json         # Машиночитаемый реестр всех документов и версий
│
├── Product/                       # Поведение продукта и пользовательские требования
│   ├── README.md                  # Индекс раздела Product
│   ├── 01-context.md              # Цель, границы и концепция продукта
│   ├── 02-users-and-scenarios.md  # Роли пользователей и ключевые сценарии (Search, RAG, MCP, SSO, Admin)
│   ├── 03-requirements.md         # Функциональные и нефункциональные требования
│   ├── 04-interface-and-behavior.md # Спецификация интерфейсов и логики взаимодействия экранов
│   └── 05-rules-and-data-meaning.md # Семантика данных (SO дамп, связи графа, схемы прав)
│
├── Tech/                          # Техническое устройство системы
│   ├── README.md                  # Индекс раздела Tech
│   ├── 01-architecture.md         # Архитектура сервисов, потоки данных, протоколы
│   ├── 02-running.md              # Инструкции по развертыванию, конфигурация окружения
│   ├── 03-data.md                 # Форматы хранения (инвертированный индекс, БД пользователей, графы)
│   ├── 04-api-and-access.md       # Спецификация REST API, gRPC, SSO (OAuth2/OIDC), MCP
│   ├── 05-verification.md         # Методология тестирования, оценка качества поиска (Recall@k, MRR)
│   └── WORKSPACE_MAP.md           # Карта кодовой базы и ответственности модулей
│
└── Service/                       # Управленческие и архитектурные решения
    ├── README.md                  # Индекс раздела Service
    ├── decisions.md               # Журнал архитектурных решений (ADR)
    ├── sources.md                 # Источники исходных требований и методических указаний
    └── verification.md            # Отчеты и результаты проверок контрольных точек
```

---

## Навигация по ключевым сценариям

1. **Поиск ошибок и RAG-синтез:** см. [02-users-and-scenarios.md](Product/02-users-and-scenarios.md) и [01-architecture.md](Tech/01-architecture.md).
2. **Граф связанных тредов и визуализация:** см. [03-requirements.md](Product/03-requirements.md) и [03-data.md](Tech/03-data.md).
3. **Авторизация, SSO и панель администрирования:** см. [04-api-and-access.md](Tech/04-api-and-access.md) и [04-interface-and-behavior.md](Product/04-interface-and-behavior.md).
4. **Межсервисные контракты Protobuf:** см. каталог `protos/` (`search.proto`, `graph.proto`).
