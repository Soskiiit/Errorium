/**
 * Errorium Single Page Application (Pure Vanilla JS, Zero Frameworks)
 * Implements chat, search, drawer navigation, MCP modal, and responsive interactions
 */

(function () {
    'use strict';

    // =========================================================================
    // Chat State & Real API Integration (Wave C - Task C2)
    // =========================================================================
    let currentChatId = null;
    let chatsList = [];
    let pendingResetToken = null;

    // =========================================================================
    // Multi-Language Localization System (EN <-> RU)
    // Translates all interface text dynamically except 'Errorium'
    // =========================================================================
    const I18N_STRINGS = {
        en: {
            "Недавние чаты": "Recent chats",
            "Новый чат": "New chat",
            "О нас": "About us",
            "Войти": "Log in",
            "Выйти": "Log out",
            "Log out": "Log out",
            "Встречайте новые функции!": "Meet new features!",
            "Бэкенд-разработчик": "Backend developer",
            "Фронтенд-разработчик": "Frontend developer",
            "Фуллстек-разработчик": "Fullstack developer",
            "Фулстек-разработчик": "Fullstack developer",
            "DevOps-инженер": "DevOps engineer",
            "QA-инженер": "QA engineer",
            "Data engineer / ML": "Data engineer / ML",
            "Инженер данных / ML": "Data engineer / ML",
            "Системный архитектор": "System architect",
            "Мобильный разработчик": "Mobile developer",
            "Инженер по безопасности": "Security engineer",
            "Тимлид / Техлид": "Tech lead",
            "Личные данные": "Personal information",
            "Имя пользователя": "Username",
            "Почта": "Email",
            "Роль": "Role",
            "Мужской": "Male",
            "Женский": "Female",
            "Не указано": "Not selected",
            "Безопасность": "Security",
            "Сменить пароль": "Change password",
            "Язык": "Language",
            "Английский": "English",
            "Русский": "Русский",
            "Элемент": "Element",
            "Реагирует": "Reacts",
            "на": "with",
            "разработчиков": "developers",
            "Стек": "Stack",
            "переполнен.": "overflow.",
            "Ошибка": "Issue",
            "известна.": "overknown.",
            "Что пошло не так?": "What went wrong?",
            "ИИ может допускать ошибки. Проверяйте важную информацию.": "AI can make mistakes. Check important information.",
            "Вставить полный стек-трейс": "Paste full stack trace",
            "Ошибка Python": "Python error",
            "Исключение Java / JVM": "Java / JVM exception",
            "Паника Go / runtime": "Go panic / runtime",
            "Ошибка C++ / Segfault": "C++ / Segfault",
            "Вход в Errorium": "Log in to Errorium",
            "Синхронизируйте историю ошибок, знания команды и подключайте Errorium через MCP для Cursor и Claude Code.": "Sync your recent error history, team knowledge, and connect Errorium via MCP for Cursor & Claude Code.",
            "Продолжить через GitHub": "Continue with GitHub",
            "Продолжить через Google SSO": "Continue with Google SSO",
            "Пароль": "Password",
            "Запомнить меня": "Remember me",
            "Забыли пароль?": "Forgot password?",
            "или": "or",
            "Войти с Яндекс ID": "Continue with Yandex ID",
            "Впервые здесь?": "New here?",
            "Регистрация": "Sign up",
            "Подтвердите пароль": "Confirm password",
            "Я принимаю Условия и Конфиденциальность": "I accept Terms & Privacy",
            "Создать аккаунт": "Create account",
            "Регистрация с Яндекс ID": "Sign up with Yandex ID",
            "Уже есть аккаунт?": "Already have an account?",
            "Введите почту от вашего аккаунта": "Enter your account email",
            "Отправить ссылку для сброса": "Send reset link",
            "Если этот email зарегистрирован, мы отправили ссылку для сброса.": "If this email is registered, we've sent a reset link.",
            "Не забудьте проверить папку «Спам»": "Don't forget to check your spam folder",
            "Войдите, чтобы сохранять историю чатов": "Log in to save chat history",
            "Новый пароль": "Enter a new password",
            "Повторите пароль": "Enter it again",
            "Подтвердить": "Approve",
            "Продолжить": "Continue",
            "Помните, что смена пароля завершит сеансы на всех остальных устройствах": "Keep in mind that changing your password will sign you out of all other devices",
            "Подключите поисковое ядро Errorium напрямую к Codex, Claude Code, Antigravity и DeepSeek Harness.": "Connect Errorium search engine directly to Codex, Claude Code, Antigravity, and DeepSeek Harness.",
            "Подключите поисковое ядро Errorium напрямую к Cursor, Claude Code и Windsurf.": "Connect Errorium search engine directly to Codex, Claude Code, Antigravity, and DeepSeek Harness.",
            "Ваш Remote MCP Bearer токен": "Your Remote MCP Bearer Token",
            "Копировать": "Copy",
            "Копировать JSON": "Copy JSON",
            "Об Errorium": "About Errorium",
            "Элемент 404. Вызывает реакцию разработчиков. Стек переполнен, разраб преисполнен.": "Element 404. Reacts with developers. Stack overflow. Issue overknown.",
            "— это интеллектуальная поисковая и RAG-система, созданная для нахождения точных решений сложных программных ошибок, исключений и стек-трейсов.": "is an intelligent search and RAG engine designed to find exact solutions for complex software errors, exceptions, and stack traces.",
            "Дамп Stack Overflow": "Stack Overflow Dump",
            "Реализовано с нуля (k₁=1.5, b=0.75)": "Built From Scratch (k₁=1.5, b=0.75)",
            "Без внешних поисковых систем": "Zero External Search Engines",
            "Интеграция с агентами в IDE": "IDE Agent Integration",
            "Разработано в рамках курса НИРС (ОАД) МГТУ им. Н. Э. Баумана (кафедра ИУ5).": "Developed as part of the Research (OAD) curriculum at Bauman Moscow State Technical University (IU5).",
            "Параметры и настройки": "Preferences & Parameters",
            "Поисковое ядро BM25": "BM25 Search Core",
            "Собственный инвертированный индекс (k₁ = 1.5, b = 0.75)": "Self-hosted inverted index (k₁ = 1.5, b = 0.75)",
            "Подключено (:50051)": "Connected (:50051)",
            "RAG-генерация ответов": "RAG Streaming",
            "Потоковый вывод синтезированных ответов в реальном времени через SSE": "Stream synthesized answers in real time via SSE",
            "Поиск": "Search",
            "Код решения": "Solution Code",
            "Скопировать код": "Copy code",
            "Источники Stack Overflow": "Stack Overflow Sources",
            "Ранжированные источники Stack Overflow": "Stack Overflow Sources",
            "Свернуть панель": "Collapse sidebar",
            "Развернуть панель": "Expand sidebar",
            "Закрыть меню": "Close menu",
            "Закрыть окно": "Close dialog",
            "Начать новый чат": "Start new chat",
            "Меню навигации": "Toggle navigation menu",
            "Открыть профиль / вход": "Open profile / auth",
            "Профиль пользователя и настройки": "User Profile and Settings",
            "Навигация по недавним чатам": "Recent chats navigation",
            "Отправить запрос ошибки": "Send error query",
            "Дополнительные действия и вставка стек-трейса": "Additional actions and stack trace paste",
            "Изменить аватар": "Edit avatar",
            "Дополнительно": "More options",
            "Скопировать токен": "Copy Token",
            "Скопировать JSON": "Copy JSON",
            "Скопировано в буфер": "Copied to clipboard",
            "Архитектура и исследования": "Engineering & Research",
            "Интеграция с AI": "AI Integration",
            "Назад в чат": "Back to Chat",
            "Попробовать": "Let's try",
            "Обзор": "Overview",
            "Попробовать Errorium": "Start debugging with Errorium",
            "Start debugging with Errorium": "Start debugging with Errorium",
            "Узнать больше про MCP и архитектуру": "Explore MCP & Architecture",
            "Настройка MCP": "Set up MCP",
            "Поиск решений нового поколения": "Next-Gen Engineering Search",
            "Привет, это": "Hello, this is",
            "Агент в IDE": "IDE Agent",
            "ИУ5 МГТУ": "IU5 BMSTU",
            "Remote MCP через SSE": "Remote MCP over SSE",
            "Точный поиск BM25": "Precise BM25 Retrieval",
            "Защищенный Bearer-токен": "Secure Bearer Token",
            "Персональный Remote MCP Bearer токен:": "Personal Remote MCP Bearer Token:",
            "Сверхбыстрый IPC": "Sub-millisecond IPC",
            "Потоковый синтез": "Streaming Synthesis",
            "Насыщение частоты терминов": "Term Frequency Saturation",
            "Нормализация длины документа": "Document Length Normalization",
            "Потоковая выдача без задержек": "Zero-Latency Streaming Delivery",
            "Любая сложная ошибка, segmentation fault или runtime panic в вашем терминале уже кем-то решена. Errorium индексирует миллионы проверенных обсуждений с кастомным ранжированием BM25 высокой точности, доставляя готовые решения прямо в ваш браузер и агенту в IDE.": "Every cryptic exception, segmentation fault, and runtime panic you encounter in your terminal has already been diagnosed and resolved. Errorium indexes millions of verified software engineering discussions with custom high-precision BM25 ranking, delivering ready-to-run solutions directly to your browser and IDE agent.",
            "Подключите поисковое ядро Errorium BM25 и базу знаний Stack Overflow напрямую к Codex, Claude Code, Antigravity, DeepSeek Harness и собственным AI-агентам.": "Connect the Errorium BM25 search engine and Stack Overflow knowledge base directly to Codex, Claude Code, Antigravity, DeepSeek Harness, and custom AI agents.",
            "Подключите поисковое ядро Errorium BM25 и базу знаний Stack Overflow напрямую к Cursor, Claude Code, Windsurf и собственным AI-агентам.": "Connect the Errorium BM25 search engine and Stack Overflow knowledge base directly to Codex, Claude Code, Antigravity, DeepSeek Harness, and custom AI agents.",
            "Прямой потоковый транспорт через Server-Sent Events (SSE). Не требует хранения тяжелых файлов индекса в окружении вашего редактора.": "Direct streaming transport over Server-Sent Events (SSE). No heavy local index files needed in your editor environment.",
            "Токенизатор сохраняет технические идентификаторы (исключения, C++, .NET, стек-трейсы) в исходном виде без искажения корней.": "Tokenization preserves technical identifiers (Exceptions, C++, .NET, stack traces) exactly as written without damaging stems.",
            "Каждый разработчик получает персональный токен аутентификации для приватного и высокоскоростного потокового поиска.": "Every developer receives an authenticated personal token for rate-limited, private, and high-throughput query streaming.",
            "Интеллектуальная поисковая и RAG-система, спроектированная с нуля для диагностики сложных программных ошибок, паник времени выполнения и стек-трейсов.": "An intelligent search and RAG engine engineered from first principles to diagnose complex software exceptions, runtime panics, and stack traces.",
            "Потоковый парсинг XML с самописным построением индекса SPIMI": "Streamed XML parsing with custom SPIMI index construction",
            "Top-K дистилляция документов с извлечением фрагментов кода и пояснений": "Top-K document distillation with code snippet extraction and explanations",
            "Реализовано с нуля": "Built From Scratch",
            "Формула ИУ5 МГТУ: k₁ = 1.5, b = 0.75, без готовых поисковых библиотек": "IU5 BMSTU formula: k₁ = 1.5, b = 0.75, zero third-party search libraries",
            "Высокопроизводительные контракты Protobuf между Gateway и Core (:50051)": "High-performance protobuf contracts between Gateway and Core (:50051)",
            "ИУ5 МГТУ им. Н. Э. Баумана • НИРС (ОАД)": "IU5 BMSTU • Research Coursework (OAD)",
            "Спроектировано для инженеров, подтверждено бенчмарками": "Designed for Developers, Verified by Benchmarks",
            "Errorium строго следует академическим ограничениям: полный запрет готовых поисковых систем (Elasticsearch, Solr, Lucene, rank-bm25). Инвертированный индекс, списки постингов, токенизация кода и ранжирование BM25 полностью написаны с нуля и непрерывно верифицируются метриками Recall@k и MRR.": "Errorium enforces strict academic guidelines: zero reliance on ready-made search engines (Elasticsearch, Solr, Lucene, rank-bm25). The inverted index, memory-mapped postings list, code tokenization, and BM25 ranking formulas are completely custom-built and continuously measured against Recall@k and MRR benchmarks.",
            "Карта связей": "Star Map",
            "Связанные треды Stack Overflow": "Related Stack Overflow threads",
            "Лучший результат": "Top result",
            "Результаты Top-K": "Top-K results",
            "Дубликаты": "Duplicates",
            "Ссылки в ответах": "Linked in answers",
            "Перетаскивайте узлы · Колесо — масштаб · Клик по звезде — детали": "Drag nodes · Scroll to zoom · Click a star for details",
            "Принятый ответ": "Accepted answer",
            "Лучший ответ": "Best answer",
            "Открыть на Stack Overflow": "Open on Stack Overflow",
            "Открыть карту связей": "Open Star Map",
            "Приблизить": "Zoom in",
            "Отдалить": "Zoom out",
            "Вписать граф": "Fit graph",
            "Закрыть детали": "Close details",
            "голосов": "votes",
            "Тред-дубликат": "Duplicate thread",
            "Связанный тред": "Linked thread",
            "Результат Top-K": "Top-K result"
        },
        ru: {
            "Recent chats": "Недавние чаты",
            "New chat": "Новый чат",
            "About us": "О нас",
            "Log in": "Войти",
            "Log out": "Выйти",
            "Выйти": "Выйти",
            "Meet new features!": "Встречайте новые функции!",
            "Backend developer": "Бэкенд-разработчик",
            "Frontend developer": "Фронтенд-разработчик",
            "Fullstack developer": "Фуллстек-разработчик",
            "DevOps engineer": "DevOps-инженер",
            "QA engineer": "QA-инженер",
            "Data engineer / ML": "Инженер данных / ML",
            "System architect": "Системный архитектор",
            "Mobile developer": "Мобильный разработчик",
            "Security engineer": "Инженер по безопасности",
            "Tech lead": "Тимлид / Техлид",
            "Personal information": "Личные данные",
            "Username": "Имя пользователя",
            "Email": "Почта",
            "Role": "Роль",
            "Male": "Мужской",
            "Female": "Женский",
            "Not selected": "Не указано",
            "Not specified": "Не указано",
            "Security": "Безопасность",
            "Change password": "Сменить пароль",
            "Language": "Язык",
            "English": "Английский",
            "Русский": "Русский",
            "Element": "Элемент",
            "Reacts": "Реагирует",
            "with": "на",
            "developers": "разработчиков",
            "Stack": "Стек",
            "overflow.": "переполнен.",
            "Issue": "Ошибка",
            "overknown.": "известна.",
            "What went wrong?": "Что пошло не так?",
            "AI can make mistakes. Check important information.": "ИИ может допускать ошибки. Проверяйте важную информацию.",
            "Paste full stack trace": "Вставить полный стек-трейс",
            "Python error": "Ошибка Python",
            "Java / JVM exception": "Исключение Java / JVM",
            "Go panic / runtime": "Паника Go / runtime",
            "C++ / Segfault": "Ошибка C++ / Segfault",
            "Log in to Errorium": "Вход в Errorium",
            "Sync your recent error history, team knowledge, and connect Errorium via MCP for Cursor & Claude Code.": "Синхронизируйте историю ошибок, знания команды и подключайте Errorium через MCP для Cursor и Claude Code.",
            "Continue with GitHub": "Продолжить через GitHub",
            "Continue with Google SSO": "Продолжить через Google SSO",
            "Password": "Пароль",
            "Remember me": "Запомнить меня",
            "Forgot password?": "Забыли пароль?",
            "or": "или",
            "Continue with Yandex ID": "Войти с Яндекс ID",
            "New here?": "Впервые здесь?",
            "Sign up": "Регистрация",
            "Confirm password": "Подтвердите пароль",
            "I accept Terms & Privacy": "Я принимаю Условия и Конфиденциальность",
            "Create account": "Создать аккаунт",
            "Sign up with Yandex ID": "Регистрация с Яндекс ID",
            "Already have an account?": "Уже есть аккаунт?",
            "Enter your account email": "Введите почту от вашего аккаунта",
            "Send reset link": "Отправить ссылку для сброса",
            "If this email is registered, we've sent a reset link.": "Если этот email зарегистрирован, мы отправили ссылку для сброса.",
            "Don't forget to check your spam folder": "Не забудьте проверить папку «Спам»",
            "Log in to save chat history": "Войдите, чтобы сохранять историю чатов",
            "Enter a new password": "Новый пароль",
            "Enter it again": "Повторите пароль",
            "Approve": "Подтвердить",
            "Continue": "Продолжить",
            "Keep in mind that changing your password will sign you out of all other devices": "Помните, что смена пароля завершит сеансы на всех остальных устройствах",
            "Connect Errorium search engine directly to Codex, Claude Code, Antigravity, and DeepSeek Harness.": "Подключите поисковое ядро Errorium напрямую к Codex, Claude Code, Antigravity и DeepSeek Harness.",
            "Connect Errorium search engine directly to Cursor, Claude Code, and Windsurf.": "Подключите поисковое ядро Errorium напрямую к Codex, Claude Code, Antigravity и DeepSeek Harness.",
            "Your Remote MCP Bearer Token": "Ваш Remote MCP Bearer токен",
            "Copy": "Копировать",
            "Copy JSON": "Копировать JSON",
            "About Errorium": "Об Errorium",
            "Element 404. Reacts with developers. Stack overflow. Issue overknown.": "Элемент 404. Вызывает реакцию разработчиков. Стек переполнен, разраб преисполнен.",
            "is an intelligent search and RAG engine designed to find exact solutions for complex software errors, exceptions, and stack traces.": "— это интеллектуальная поисковая и RAG-система, созданная для нахождения точных решений сложных программных ошибок, исключений и стек-трейсов.",
            "Stack Overflow Dump": "Дамп Stack Overflow",
            "Built From Scratch (k₁=1.5, b=0.75)": "Реализовано с нуля (k₁=1.5, b=0.75)",
            "Zero External Search Engines": "Без внешних поисковых систем",
            "IDE Agent Integration": "Интеграция с агентами в IDE",
            "Developed as part of the Research (OAD) curriculum at Bauman Moscow State Technical University (IU5).": "Разработано в рамках курса НИРС (ОАД) МГТУ им. Н. Э. Баумана (кафедра ИУ5).",
            "Preferences & Parameters": "Параметры и настройки",
            "BM25 Search Core": "Поисковое ядро BM25",
            "Self-hosted inverted index (k₁ = 1.5, b = 0.75)": "Собственный инвертированный индекс (k₁ = 1.5, b = 0.75)",
            "Connected (:50051)": "Подключено (:50051)",
            "RAG Streaming": "RAG-генерация ответов",
            "Stream synthesized answers in real time via SSE": "Потоковый вывод синтезированных ответов в реальном времени через SSE",
            "Search": "Поиск",
            "Solution Code": "Код решения",
            "Copy code": "Скопировать код",
            "Stack Overflow Sources": "Источники Stack Overflow",
            "Ranked Stack Overflow Sources": "Источники Stack Overflow",
            "Collapse sidebar": "Свернуть панель",
            "Expand sidebar": "Развернуть панель",
            "Close menu": "Закрыть меню",
            "Close dialog": "Закрыть окно",
            "Start new chat": "Начать новый чат",
            "Toggle navigation menu": "Меню навигации",
            "Open profile / auth": "Открыть профиль / вход",
            "User Profile and Settings": "Профиль пользователя и настройки",
            "Recent chats navigation": "Навигация по недавним чатам",
            "Send error query": "Отправить запрос ошибки",
            "Additional actions and stack trace paste": "Дополнительные действия и вставка стек-трейса",
            "Edit avatar": "Изменить аватар",
            "More options": "Дополнительно",
            "Copy Token": "Скопировать токен",
            "Copy JSON": "Скопировать JSON",
            "Engineering & Research": "Архитектура и исследования",
            "AI Integration": "Интеграция с AI",
            "IDE Agent": "Агент в IDE",
            "IU5 BMSTU": "ИУ5 МГТУ",
            "About Errorium": "Об Errorium",
            "Remote MCP over SSE": "Remote MCP через SSE",
            "Precise BM25 Retrieval": "Точный поиск BM25",
            "Secure Bearer Token": "Защищенный Bearer-токен",
            "Personal Remote MCP Bearer Token:": "Персональный Remote MCP Bearer токен:",
            "Sub-millisecond IPC": "Сверхбыстрый IPC",
            "Streaming Synthesis": "Потоковый синтез",
            "Term Frequency Saturation": "Насыщение частоты терминов",
            "Document Length Normalization": "Нормализация длины документа",
            "Zero-Latency Streaming Delivery": "Потоковая выдача без задержек",
            "Back to Chat": "Назад в чат",
            "Let's try": "Попробовать",
            "Overview": "Обзор",
            "Try Errorium": "Попробовать Errorium",
            "Start debugging with Errorium": "Попробовать Errorium",
            "Explore MCP & Architecture": "Узнать больше про MCP и архитектуру",
            "Set up MCP": "Настройка MCP",
            "Next-Gen Engineering Search": "Поиск решений нового поколения",
            "Hello, this is": "Привет, это",
            "Every cryptic exception, segmentation fault, and runtime panic you encounter in your terminal has already been diagnosed and resolved. Errorium indexes millions of verified software engineering discussions with custom high-precision BM25 ranking, delivering ready-to-run solutions directly to your browser and IDE agent.": "Любая сложная ошибка, segmentation fault или runtime panic в вашем терминале уже кем-то решена. Errorium индексирует миллионы проверенных обсуждений с кастомным ранжированием BM25 высокой точности, доставляя готовые решения прямо в ваш браузер и агенту в IDE.",
            "Connect the Errorium BM25 search engine and Stack Overflow knowledge base directly to Codex, Claude Code, Antigravity, DeepSeek Harness, and custom AI agents.": "Подключите поисковое ядро Errorium BM25 и базу знаний Stack Overflow напрямую к Codex, Claude Code, Antigravity, DeepSeek Harness и собственным AI-агентам.",
            "Connect the Errorium BM25 search engine and Stack Overflow knowledge base directly to Cursor, Claude Code, Windsurf, and custom AI agents.": "Подключите поисковое ядро Errorium BM25 и базу знаний Stack Overflow напрямую к Codex, Claude Code, Antigravity, DeepSeek Harness и собственным AI-агентам.",
            "Direct streaming transport over Server-Sent Events (SSE). No heavy local index files needed in your editor environment.": "Прямой потоковый транспорт через Server-Sent Events (SSE). Не требует хранения тяжелых файлов индекса в окружении вашего редактора.",
            "Tokenization preserves technical identifiers (Exceptions, C++, .NET, stack traces) exactly as written without damaging stems.": "Токенизатор сохраняет технические идентификаторы (исключения, C++, .NET, стек-трейсы) в исходном виде без искажения корней.",
            "Every developer receives an authenticated personal token for rate-limited, private, and high-throughput query streaming.": "Каждый разработчик получает персональный токен аутентификации для приватного и высокоскоростного потокового поиска.",
            "An intelligent search and RAG engine engineered from first principles to diagnose complex software exceptions, runtime panics, and stack traces.": "Интеллектуальная поисковая и RAG-система, спроектированная с нуля для диагностики сложных программных ошибок, паник времени выполнения и стек-трейсов.",
            "Streamed XML parsing with custom SPIMI index construction": "Потоковый парсинг XML с самописным построением индекса SPIMI",
            "Top-K document distillation with code snippet extraction and explanations": "Top-K дистилляция документов с извлечением фрагментов кода и пояснений",
            "Built From Scratch": "Реализовано с нуля",
            "IU5 BMSTU formula: k₁ = 1.5, b = 0.75, zero third-party search libraries": "Формула ИУ5 МГТУ: k₁ = 1.5, b = 0.75, без готовых поисковых библиотек",
            "High-performance protobuf contracts between Gateway and Core (:50051)": "Высокопроизводительные контракты Protobuf между Gateway и Core (:50051)",
            "IU5 BMSTU • Research Coursework (OAD)": "ИУ5 МГТУ им. Н. Э. Баумана • НИРС (ОАД)",
            "Designed for Developers, Verified by Benchmarks": "Спроектировано для инженеров, подтверждено бенчмарками",
            "Errorium enforces strict academic guidelines: zero reliance on ready-made search engines (Elasticsearch, Solr, Lucene, rank-bm25). The inverted index, memory-mapped postings list, code tokenization, and BM25 ranking formulas are completely custom-built and continuously measured against Recall@k and MRR benchmarks.": "Errorium строго следует академическим ограничениям: полный запрет готовых поисковых систем (Elasticsearch, Solr, Lucene, rank-bm25). Инвертированный индекс, списки постингов, токенизация кода и ранжирование BM25 полностью написаны с нуля и непрерывно верифицируются метриками Recall@k и MRR.",
            "Star Map": "Карта связей",
            "Back to chat": "Назад в чат",
            "Related Stack Overflow threads": "Связанные треды Stack Overflow",
            "Top result": "Лучший результат",
            "Top-K results": "Результаты Top-K",
            "Duplicates": "Дубликаты",
            "Linked in answers": "Ссылки в ответах",
            "Drag nodes · Scroll to zoom · Click a star for details": "Перетаскивайте узлы · Колесо — масштаб · Клик по звезде — детали",
            "Accepted answer": "Принятый ответ",
            "Best answer": "Лучший ответ",
            "Open on Stack Overflow": "Открыть на Stack Overflow",
            "Open Star Map": "Открыть карту связей",
            "Zoom in": "Приблизить",
            "Zoom out": "Отдалить",
            "Fit graph": "Вписать граф",
            "Close details": "Закрыть детали",
            "votes": "голосов",
            "Duplicate thread": "Тред-дубликат",
            "Linked thread": "Связанный тред",
            "Top-K result": "Результат Top-K"
        }
    };

    function t(str, targetLang = currentLang) {
        if (!str || typeof str !== 'string') return str;
        const trimmed = str.trim();
        if (targetLang === 'ru') {
            return I18N_STRINGS.ru[trimmed] || trimmed;
        } else {
            return I18N_STRINGS.en[trimmed] || trimmed;
        }
    }

    const APP_LANG_KEY = 'errorium_app_language';
    let currentLang = 'English';

    function getCurrentLanguage() {
        const storedLang = localStorage.getItem(APP_LANG_KEY);
        if (storedLang) return storedLang;
        const user = getAuthUser();
        return (user && user.language) ? user.language : currentLang;
    }

    function applyLanguage(lang) {
        currentLang = (lang === 'Русский' || lang === 'ru') ? 'Русский' : 'English';
        localStorage.setItem(APP_LANG_KEY, currentLang);
        const isRu = currentLang === 'Русский';
        const target = isRu ? 'ru' : 'en';

        // 1. Walk DOM text nodes and translate phrases
        const walker = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT, null, false);
        let node;
        while ((node = walker.nextNode())) {
            const parent = node.parentElement;
            if (!parent || parent.tagName === 'SCRIPT' || parent.tagName === 'STYLE') continue;
            if (parent.closest('code') || parent.closest('pre')) continue;
            if (parent.classList && (parent.classList.contains('header-logo') || parent.classList.contains('sidebar-brand') || parent.classList.contains('auth-brand-heading'))) continue;

            const original = node.nodeValue.trim();
            if (original && original !== 'Errorium') {
                const translated = t(original, target);
                if (translated && translated !== original) {
                    node.nodeValue = node.nodeValue.replace(original, translated);
                }
            }
        }

        // 2. Translate placeholders
        document.querySelectorAll('input, textarea').forEach(input => {
            const ph = input.getAttribute('placeholder');
            if (ph && ph !== 'Errorium') {
                const translated = t(ph, target);
                if (translated) input.setAttribute('placeholder', translated);
            }
        });

        // 3. Translate titles and aria-labels
        document.querySelectorAll('[title]').forEach(el => {
            const val = el.getAttribute('title');
            if (val && val !== 'Errorium') {
                const translated = t(val, target);
                if (translated) el.setAttribute('title', translated);
            }
        });

        document.querySelectorAll('[aria-label]').forEach(el => {
            const val = el.getAttribute('aria-label');
            if (val && val !== 'Errorium') {
                const translated = t(val, target);
                if (translated) el.setAttribute('aria-label', translated);
            }
        });

        // 4. Update language display row in profile & landing header
        if (elements.profileInfoLang) {
            elements.profileInfoLang.textContent = isRu ? 'Русский' : 'English';
        }
        document.querySelectorAll('#lang-dropdown-popover .profile-dropdown-item').forEach(item => {
            if (item.getAttribute('data-lang') === currentLang) {
                item.classList.add('active');
            } else {
                item.classList.remove('active');
            }
        });

        const landingLangCurrent = document.getElementById('landing-lang-current');
        if (landingLangCurrent) {
            landingLangCurrent.textContent = isRu ? 'Русский' : 'English';
        }
        document.querySelectorAll('.landing-lang-option').forEach(item => {
            if (item.getAttribute('data-lang') === currentLang) {
                item.classList.add('active');
            } else {
                item.classList.remove('active');
            }
        });

        // 5. Update Center Hero Slogan
        const heroPrimary = document.querySelector('.hero-primary-text');
        const heroSecondary = document.querySelector('.hero-secondary-text');
        if (heroPrimary && heroSecondary) {
            if (isRu) {
                heroPrimary.innerHTML = 'Элемент<span class="space"> </span><span class="error-code-red">404.</span><span class="space"> </span>Вызывает<span class="space"> </span>реакцию<span class="space"> </span>разработчиков.';
                heroSecondary.innerHTML = 'Стек<span class="space"> </span>переполнен,<span class="space"> </span>разраб<span class="space"> </span>преисполнен.';
            } else {
                heroPrimary.innerHTML = 'Element<span class="space"> </span><span class="error-code-red">404.</span><span class="space"> </span>Reacts<span class="space"> </span>with<span class="space"> </span>developers';
                heroSecondary.innerHTML = 'Stack<span class="space"> </span>overflow.<span class="space"> </span>Issue<span class="space"> </span>overknown.';
            }
        }

        // 5.1 Update Landing Slogan (English / Russian)
        const landingTagline = document.getElementById('element-slogan-tagline');
        if (landingTagline) {
            if (isRu) {
                landingTagline.innerHTML = `
                    <span class="landing-slogan-line">Стек переполнен, <strong class="tagline-highlight">разраб преисполнен</strong> — поиск готовых инженерных решений</span>
                    <span class="landing-slogan-line">и RAG-синтез по 100 ГБ базе знаний.</span>
                `;
            } else {
                landingTagline.innerHTML = `
                    <span class="landing-slogan-line">Stack overflow. <strong class="tagline-highlight">Issue overknown</strong> — instant engineering solutions &amp; RAG synthesis</span>
                    <span class="landing-slogan-line">across 100 GB knowledge base.</span>
                `;
            }
        }

        // 6. Update user role in sidebar & profile
        const activeUser = getAuthUser();
        const roleVal = (activeUser && activeUser.role) ? activeUser.role : 'Backend developer';
        const localizedRole = t(roleVal, target);
        if (elements.sidebarAuthRole) {
            elements.sidebarAuthRole.textContent = localizedRole;
        }
        if (elements.profileInfoRole) {
            elements.profileInfoRole.textContent = localizedRole;
        }

        // 7. Update Landing intro description (with highlighted technical terms)
        const landingIntroDesc = document.querySelector('.landing-intro-description');
        if (landingIntroDesc) {
            if (isRu) {
                landingIntroDesc.innerHTML = 'Любая сложная ошибка, <span class="tech-highlight">segmentation fault</span> или <span class="tech-highlight">runtime panic</span> в вашем терминале уже кем-то решена. Errorium индексирует миллионы проверенных обсуждений с кастомным ранжированием <span class="tech-highlight">BM25</span> высокой точности, доставляя готовые решения прямо в ваш браузер и агенту в <span class="tech-highlight">IDE</span>.';
            } else {
                landingIntroDesc.innerHTML = 'Every cryptic exception, <span class="tech-highlight">segmentation fault</span>, and <span class="tech-highlight">runtime panic</span> you encounter in your terminal has already been diagnosed and resolved. Errorium indexes millions of verified software engineering discussions with custom high-precision <span class="tech-highlight">BM25</span> ranking, delivering ready-to-run solutions directly to your browser and <span class="tech-highlight">IDE</span> agent.';
            }
        }
    }

    // =========================================================================
    // DOM Elements Cache
    // =========================================================================
    const elements = {
        app: document.getElementById('app'),
        backdrop: document.getElementById('backdrop'),
        sidebar: document.getElementById('sidebar'),
        sidebarBrand: document.getElementById('sidebar-brand'),
        btnMenu: document.getElementById('btn-menu'),
        btnToggleSidebar: document.getElementById('btn-toggle-sidebar'),
        btnExpandSidebar: document.getElementById('btn-expand-sidebar'),
        btnCloseSidebar: document.getElementById('btn-close-sidebar'),
        headerLogo: document.getElementById('header-logo'),
        btnProfile: document.getElementById('btn-profile'),
        btnLoginRow: document.getElementById('btn-login-row'),
        btnNewChat: document.getElementById('btn-new-chat'),
        btnMcp: document.getElementById('btn-mcp'),
        btnAbout: document.getElementById('btn-about'),
        btnSettings: document.getElementById('btn-settings'),

        // Authorized Sidebar & Header
        sidebarUserGuest: document.getElementById('sidebar-user-guest'),
        sidebarUserAuth: document.getElementById('sidebar-user-auth'),
        btnAuthUserRow: document.getElementById('btn-auth-user-row'),
        sidebarAuthAvatar: document.getElementById('sidebar-auth-avatar'),
        sidebarAuthName: document.getElementById('sidebar-auth-name'),
        sidebarAuthRole: document.getElementById('sidebar-auth-role'),
        btnSettingsAuth: document.getElementById('btn-settings-auth'),
        headerAvatarGuest: document.getElementById('header-avatar-guest'),
        headerAvatarAuth: document.getElementById('header-avatar-auth'),

        // Profile Screen Elements
        profileView: document.getElementById('profile-view'),
        btnProfileBack: document.getElementById('btn-profile-back'),
        profileAvatarLarge: document.getElementById('profile-avatar-large'),
        profileAvatarLetter: document.getElementById('profile-avatar-letter'),
        profileAvatarImg: document.getElementById('profile-avatar-img'),
        profileAvatarEditBadge: document.getElementById('profile-avatar-edit-badge'),
        profileAvatarFileInput: document.getElementById('profile-avatar-file-input'),
        profileUsername: document.getElementById('profile-username'),
        profileInfoUsername: document.getElementById('profile-info-username'),
        profileInfoEmail: document.getElementById('profile-info-email'),
        profileInfoRole: document.getElementById('profile-info-role'),
        profileInfoRoleRow: document.getElementById('profile-info-role-row'),
        roleDropdownPopover: document.getElementById('role-dropdown-popover'),
        profileRoleChevron: document.getElementById('profile-role-chevron'),
        profileInfoGender: document.getElementById('profile-info-gender'),
        profileInfoGenderRow: document.getElementById('profile-info-gender-row'),
        genderDropdownPopover: document.getElementById('gender-dropdown-popover'),
        profileGenderChevron: document.getElementById('profile-gender-chevron'),
        profileRowSecurity: document.getElementById('profile-row-security'),
        profileInfoLang: document.getElementById('profile-info-lang'),
        profileInfoLangRow: document.getElementById('profile-info-lang-row'),
        langDropdownPopover: document.getElementById('lang-dropdown-popover'),
        profileLangChevron: document.getElementById('profile-lang-chevron'),

        searchRecentChats: document.getElementById('search-recent-chats'),
        recentChatsList: document.getElementById('recent-chats-list'),
        recentChatsToggle: document.getElementById('recent-chats-toggle'),
        recentChatsContainer: document.getElementById('recent-chats-container'),

        heroView: document.getElementById('hero-view'),
        landingView: document.getElementById('landing-view'),
        btnLandingBack: document.getElementById('btn-landing-back'),
        btnLandingCta: document.getElementById('btn-landing-cta'),
        chatView: document.getElementById('chat-view'),
        messagesContainer: document.getElementById('messages-container'),
        promptFooter: document.getElementById('app-prompt-footer'),
        promptBarContainer: document.querySelector('.prompt-bar-container'),

        // Star Map (FR-3: interactive graph of related SO threads)
        starMapView: document.getElementById('star-map-view'),
        starMapCanvas: document.getElementById('star-map-canvas'),
        btnStarMapBack: document.getElementById('btn-star-map-back'),
        starMapSubtitle: document.getElementById('star-map-subtitle'),
        starMapTooltip: document.getElementById('star-map-tooltip'),
        starMapHint: document.getElementById('star-map-hint'),
        starMapDetailPanel: document.getElementById('star-map-detail-panel'),
        btnStarMapDetailClose: document.getElementById('btn-star-map-detail-close'),
        starMapDetailCategory: document.getElementById('star-map-detail-category'),
        starMapDetailTitle: document.getElementById('star-map-detail-title'),
        starMapDetailTags: document.getElementById('star-map-detail-tags'),
        starMapDetailAnswerLabel: document.getElementById('star-map-detail-answer-label'),
        starMapDetailAnswer: document.getElementById('star-map-detail-answer'),
        starMapDetailLink: document.getElementById('star-map-detail-link'),
        btnStarMapZoomIn: document.getElementById('btn-star-map-zoom-in'),
        btnStarMapZoomOut: document.getElementById('btn-star-map-zoom-out'),
        btnStarMapFit: document.getElementById('btn-star-map-fit'),

        promptInput: document.getElementById('prompt-input'),
        btnSend: document.getElementById('btn-send'),
        btnAttach: document.getElementById('btn-attach'),
        attachPopover: document.getElementById('attach-popover'),
        tagsPopover: document.getElementById('tags-popover'),

        // Modals
        modalAuth: document.getElementById('modal-auth'),
        modalMcp: document.getElementById('modal-mcp'),
        modalAbout: document.getElementById('modal-about'),
        modalSettings: document.getElementById('modal-settings'),
        settingCoreStatus: document.getElementById('setting-core-status'),
        settingRagDesc: document.getElementById('setting-rag-desc'),
        settingRagProviderBadge: document.getElementById('setting-rag-provider-badge'),
        userDisplayName: document.getElementById('user-display-name'),

        // Reference Modals (services/frontend/references/)
        modalAuthRef: document.getElementById('modal-auth-reference'),
        refAuthForm: document.getElementById('ref-auth-form'),
        refAuthEmail: document.getElementById('ref-auth-email'),
        refAuthPassword: document.getElementById('ref-auth-password'),
        refAuthRemember: document.getElementById('ref-auth-remember'),
        refAuthForgot: document.getElementById('ref-auth-forgot'),
        refBtnLogin: document.getElementById('ref-btn-login'),
        refBtnYandex: document.getElementById('ref-btn-yandex'),
        refLinkSignup: document.getElementById('ref-link-signup'),

        modalRegisterRef: document.getElementById('modal-register-reference'),
        refRegisterForm: document.getElementById('ref-register-form'),
        refRegisterUsername: document.getElementById('ref-register-username'),
        refRegisterEmail: document.getElementById('ref-register-email'),
        refRegisterPassword: document.getElementById('ref-register-password'),
        refRegisterConfirm: document.getElementById('ref-register-confirm'),
        refRegisterTerms: document.getElementById('ref-register-terms'),
        refBtnRegister: document.getElementById('ref-btn-register'),
        refBtnRegisterYandex: document.getElementById('ref-btn-register-yandex'),
        refLinkSignin: document.getElementById('ref-link-signin'),

        modalChangePasswordEmail: document.getElementById('modal-change-password-email'),
        refResetEmailForm: document.getElementById('ref-reset-email-form'),
        refResetEmail: document.getElementById('ref-reset-email'),
        refBtnSendReset: document.getElementById('ref-btn-send-reset'),
        refResetNotice: document.getElementById('ref-reset-notice'),

        modalChangePassword: document.getElementById('modal-change-password'),
        refChangePasswordForm: document.getElementById('ref-change-password-form'),
        refNewPassword: document.getElementById('ref-new-password'),
        refConfirmPassword: document.getElementById('ref-confirm-password'),
        refPasswordError: document.getElementById('ref-password-error'),
        refBtnApprovePassword: document.getElementById('ref-btn-approve-password'),

        modalResetPasswordAlert: document.getElementById('modal-reset-password-alert'),
        refBtnAlertSendReset: document.getElementById('ref-btn-alert-send-reset'),

        btnCopyUrl: document.getElementById('btn-copy-url'),
        btnCopyToken: document.getElementById('btn-copy-url') || document.getElementById('btn-copy-token'),
        btnCopyMcpJson: document.getElementById('btn-copy-mcp-json'),
        mcpUrlDisplay: document.getElementById('mcp-url-display'),
        mcpTokenDisplay: document.getElementById('mcp-url-display') || document.getElementById('mcp-token-display'),
        mcpJsonSnippet: document.getElementById('mcp-json-snippet'),

        btnLandingCopyUrl: document.getElementById('btn-landing-copy-url'),
        btnLandingCopyToken: document.getElementById('btn-landing-copy-url') || document.getElementById('btn-landing-copy-token'),
        btnLandingCopyJson: document.getElementById('btn-landing-copy-json'),
        landingMcpUrl: document.getElementById('landing-mcp-url'),
        landingMcpToken: document.getElementById('landing-mcp-url') || document.getElementById('landing-mcp-token'),
        landingMcpJsonSnippet: document.getElementById('landing-mcp-json-snippet'),

        btnLoginGithub: document.getElementById('btn-login-github'),
        btnLoginGoogle: document.getElementById('btn-login-google'),

        btnSidebarLogout: document.getElementById('btn-sidebar-logout'),
        btnProfileLogout: document.getElementById('btn-profile-logout'),
        profileRowLogout: document.getElementById('profile-row-logout'),

        toast: document.getElementById('toast')
    };

    // =========================================================================
    // Image Dominant Color Extractor (Canvas sampling with saturation boost)
    // =========================================================================
    function extractDominantColor(dataUrl, callback) {
        const img = new Image();
        img.crossOrigin = 'Anonymous';
        img.onload = function () {
            try {
                const canvas = document.createElement('canvas');
                const ctx = canvas.getContext('2d');
                const size = 32; // Sample resolution
                canvas.width = size;
                canvas.height = size;
                ctx.drawImage(img, 0, 0, size, size);
                const imgData = ctx.getImageData(0, 0, size, size).data;

                let rTotal = 0, gTotal = 0, bTotal = 0, count = 0;
                let maxSat = -1, bestColor = null;

                for (let i = 0; i < imgData.length; i += 4) {
                    const r = imgData[i];
                    const g = imgData[i + 1];
                    const b = imgData[i + 2];
                    const a = imgData[i + 3];

                    // Skip transparent, near-black or near-white pixels
                    if (a < 128) continue;
                    const brightness = 0.299 * r + 0.587 * g + 0.114 * b;
                    if (brightness < 25 || brightness > 240) continue;

                    // Measure saturation
                    const max = Math.max(r, g, b);
                    const min = Math.min(r, g, b);
                    const sat = max === 0 ? 0 : (max - min) / max;

                    if (sat > maxSat) {
                        maxSat = sat;
                        bestColor = { r, g, b };
                    }

                    rTotal += r;
                    gTotal += g;
                    bTotal += b;
                    count++;
                }

                if (bestColor && maxSat > 0.2) {
                    callback(`rgba(${bestColor.r}, ${bestColor.g}, ${bestColor.b}, 0.75)`);
                } else if (count > 0) {
                    const rAvg = Math.round(rTotal / count);
                    const gAvg = Math.round(gTotal / count);
                    const bAvg = Math.round(bTotal / count);
                    callback(`rgba(${rAvg}, ${gAvg}, ${bAvg}, 0.75)`);
                } else {
                    callback('rgba(155, 81, 224, 0.65)');
                }
            } catch (err) {
                callback('rgba(155, 81, 224, 0.65)');
            }
        };
        img.onerror = function () {
            callback('rgba(155, 81, 224, 0.65)');
        };
        img.src = dataUrl;
    }

    // =========================================================================
    // Sidebar Drawer & Desktop Collapse Controls
    // =========================================================================
    const SIDEBAR_STATE_KEY = 'errorium_sidebar_state';

    // Clear legacy boolean key if present to prevent accidental default collapsed state
    try {
        localStorage.removeItem('errorium_sidebar_collapsed');
    } catch (e) {}

    function isWideScreen() {
        return window.innerWidth >= 768;
    }

    function openSidebar() {
        if (isWideScreen()) {
            expandDesktopSidebar(true);
            return;
        }
        elements.sidebar.classList.add('open');
        elements.backdrop.classList.add('active');
        document.body.style.overflow = 'hidden';
    }

    function closeSidebar() {
        elements.sidebar.classList.remove('open');
        elements.backdrop.classList.remove('active');
        document.body.style.overflow = '';
    }

    function collapseDesktopSidebar(saveState = true) {
        document.documentElement.classList.add('sidebar-collapsed');
        elements.app.classList.add('sidebar-collapsed');
        elements.sidebar.classList.add('collapsed');
        if (saveState) {
            try {
                localStorage.setItem(SIDEBAR_STATE_KEY, 'collapsed');
            } catch (e) {}
        }
    }

    function expandDesktopSidebar(saveState = true) {
        document.documentElement.classList.remove('sidebar-collapsed');
        elements.app.classList.remove('sidebar-collapsed');
        elements.sidebar.classList.remove('collapsed');
        if (saveState) {
            try {
                localStorage.setItem(SIDEBAR_STATE_KEY, 'expanded');
            } catch (e) {}
        }
    }

    function toggleDesktopSidebar() {
        if (elements.app.classList.contains('sidebar-collapsed')) {
            expandDesktopSidebar(true);
        } else {
            collapseDesktopSidebar(true);
        }
    }

    // Default state: initially EXPANDED (open by default).
    // Only if user explicitly collapsed it and saved preference, restore collapsed state.
    try {
        const savedState = localStorage.getItem(SIDEBAR_STATE_KEY);
        if (savedState === 'collapsed' && isWideScreen()) {
            collapseDesktopSidebar(false);
        } else {
            expandDesktopSidebar(false);
        }
    } catch (e) {
        expandDesktopSidebar(false);
    }

    // Safely remove preload class after first paint to re-enable smooth interactive transitions
    function removePreload() {
        requestAnimationFrame(() => {
            setTimeout(() => {
                document.body.classList.remove('preload');
            }, 60);
        });
    }

    if (document.readyState === 'loading') {
        window.addEventListener('DOMContentLoaded', removePreload);
    } else {
        removePreload();
    }

    elements.btnToggleSidebar?.addEventListener('click', toggleDesktopSidebar);
    elements.btnExpandSidebar?.addEventListener('click', toggleDesktopSidebar);

    elements.btnMenu?.addEventListener('click', () => {
        if (isWideScreen()) {
            toggleDesktopSidebar();
        } else {
            openSidebar();
        }
    });

    elements.btnCloseSidebar?.addEventListener('click', closeSidebar);
    elements.backdrop?.addEventListener('click', () => {
        closeSidebar();
        closeAllModals();
    });

    // Debounce helper (300 ms for recent chats search)
    function debounce(fn, delay) {
        let timeoutId;
        return function (...args) {
            clearTimeout(timeoutId);
            timeoutId = setTimeout(() => fn.apply(this, args), delay);
        };
    }

    // Recent chats search filter (debounced 300 ms, API-backed)
    const debouncedSearchRecentChats = debounce(function (query) {
        const tokens = window.ErroriumAPI ? window.ErroriumAPI.getTokens() : null;
        if (tokens && tokens.access_token) {
            loadRecentChats(query);
        } else {
            const items = elements.recentChatsList ? elements.recentChatsList.querySelectorAll('.recent-chat-item') : [];
            items.forEach(item => {
                const title = item.querySelector('.chat-item-title')?.textContent.toLowerCase() || '';
                item.style.display = title.includes(query.toLowerCase()) ? 'flex' : 'none';
            });
        }
    }, 300);

    elements.searchRecentChats?.addEventListener('input', function (e) {
        debouncedSearchRecentChats(e.target.value.trim());
    });

    // Recent chats collapse toggle
    elements.recentChatsToggle?.addEventListener('click', function () {
        const isCollapsed = this.classList.toggle('collapsed');
        if (elements.recentChatsContainer) {
            elements.recentChatsContainer.style.display = isCollapsed ? 'none' : '';
        }
    });

    // =========================================================================
    // Modals Controls & Authentication State
    // =========================================================================
    const AUTH_USER_KEY = 'errorium_auth_user';

    function getAuthUser() {
        try {
            const data = localStorage.getItem(AUTH_USER_KEY);
            return data ? JSON.parse(data) : null;
        } catch (e) {
            return null;
        }
    }

    function setAuthUser(user) {
        try {
            if (user) {
                const username = user.username || '';
                const profession = user.profession || user.role || 'Backend developer';
                const normalized = {
                    id: user.id || null,
                    username: username,
                    email: user.email || '',
                    role: profession,
                    profession: profession,
                    userRole: user.role || 'user',
                    gender: user.gender || 'Not specified',
                    language: user.language || 'English',
                    avatarUrl: user.avatar_url || user.avatarUrl || null,
                    avatarLetter: (username && username[0] ? username[0] : 'U').toUpperCase(),
                    avatarGlow: user.avatarGlow || null
                };
                localStorage.setItem(AUTH_USER_KEY, JSON.stringify(normalized));
            } else {
                localStorage.removeItem(AUTH_USER_KEY);
            }
        } catch (e) {}
        updateAuthUI();
    }

    function updateAuthUI() {
        const user = getAuthUser();
        if (user) {
            loadRecentChats();
            if (elements.sidebarUserGuest) elements.sidebarUserGuest.style.display = 'none';
            if (elements.sidebarUserAuth) elements.sidebarUserAuth.style.display = 'flex';
            if (elements.headerAvatarGuest) elements.headerAvatarGuest.style.display = 'none';
            if (elements.headerAvatarAuth) {
                elements.headerAvatarAuth.style.display = 'flex';
                if (user.avatarUrl) {
                    elements.headerAvatarAuth.innerHTML = `<img src="${user.avatarUrl}" style="width:100%;height:100%;border-radius:50%;object-fit:cover;" alt="Avatar" />`;
                } else {
                    elements.headerAvatarAuth.textContent = (user.avatarLetter || user.username[0] || 'U').toUpperCase();
                }
            }

            if (elements.sidebarAuthAvatar) {
                if (user.avatarUrl) {
                    elements.sidebarAuthAvatar.innerHTML = `<img src="${user.avatarUrl}" style="width:100%;height:100%;border-radius:50%;object-fit:cover;" alt="Avatar" />`;
                } else {
                    elements.sidebarAuthAvatar.textContent = (user.avatarLetter || user.username[0] || 'U').toUpperCase();
                }
            }
            const targetLangCode = (getCurrentLanguage() === 'Русский') ? 'ru' : 'en';
            const userRoleLocalized = t(user.role || 'Backend developer', targetLangCode);
            if (elements.sidebarAuthName) elements.sidebarAuthName.textContent = user.username;
            if (elements.sidebarAuthRole) elements.sidebarAuthRole.textContent = userRoleLocalized;

            if (elements.profileAvatarLetter) {
                elements.profileAvatarLetter.textContent = (user.avatarLetter || user.username[0] || 'U').toUpperCase();
            }
            if (elements.profileAvatarImg) {
                if (user.avatarUrl) {
                    elements.profileAvatarImg.src = user.avatarUrl;
                    elements.profileAvatarImg.style.display = 'block';
                    if (elements.profileAvatarLetter) elements.profileAvatarLetter.style.display = 'none';
                    if (user.avatarGlow) {
                        elements.profileAvatarLarge?.style.setProperty('--avatar-glow-color', user.avatarGlow);
                    }
                } else {
                    elements.profileAvatarImg.style.display = 'none';
                    if (elements.profileAvatarLetter) elements.profileAvatarLetter.style.display = '';
                    elements.profileAvatarLarge?.style.removeProperty('--avatar-glow-color');
                }
            }
            if (elements.profileUsername) elements.profileUsername.textContent = user.username;
            if (elements.profileInfoUsername) {
                elements.profileInfoUsername.value = user.username;
                elements.profileInfoUsername.textContent = user.username;
            }
            if (elements.profileInfoEmail) {
                elements.profileInfoEmail.value = user.email || '';
                elements.profileInfoEmail.textContent = user.email || '';
            }
            if (elements.profileInfoRole) {
                elements.profileInfoRole.textContent = userRoleLocalized;
            }
            document.querySelectorAll('.role-dropdown-item').forEach(item => {
                if (item.getAttribute('data-role') === (user.role || 'Backend developer')) {
                    item.classList.add('active');
                } else {
                    item.classList.remove('active');
                }
            });
            const currentGender = (user.gender === 'Not specified' || !user.gender) ? 'Not selected' : user.gender;
            if (elements.profileInfoGender) elements.profileInfoGender.textContent = currentGender;
            document.querySelectorAll('#gender-dropdown-popover .profile-dropdown-item').forEach(item => {
                if (item.getAttribute('data-gender') === currentGender) {
                    item.classList.add('active');
                } else {
                    item.classList.remove('active');
                }
            });

            if (elements.profileInfoLang) elements.profileInfoLang.textContent = user.language || 'English';
            document.querySelectorAll('#lang-dropdown-popover .profile-dropdown-item').forEach(item => {
                if (item.getAttribute('data-lang') === (user.language || 'English')) {
                    item.classList.add('active');
                } else {
                    item.classList.remove('active');
                }
            });
        } else {
            if (elements.sidebarUserGuest) elements.sidebarUserGuest.style.display = 'flex';
            if (elements.sidebarUserAuth) elements.sidebarUserAuth.style.display = 'none';
            if (elements.headerAvatarGuest) elements.headerAvatarGuest.style.display = 'inline-block';
            if (elements.headerAvatarAuth) elements.headerAvatarAuth.style.display = 'none';
            currentChatId = null;
            renderRecentChats([]);
        }
        applyLanguage(getCurrentLanguage());
    }

    // =========================================================================
    // SPA Routing (History API & popstate)
    // =========================================================================
    let isNavigatingFromPopState = false;

    function navigateTo(path, updateHistory = true) {
        if (updateHistory && window.location.pathname !== path) {
            window.history.pushState({ path }, '', path);
        }
        renderCurrentRoute();
    }

    function renderCurrentRoute() {
        const path = window.location.pathname;

        if (path === '/auth/callback') {
            handleAuthCallback();
            return;
        }

        if (path === '/profile') {
            if (getAuthUser()) {
                switchToProfileView(false);
            } else {
                switchToHeroView(false);
                showModal(elements.modalAuthRef || elements.modalAuth, false);
            }
            return;
        }

        if (path === '/auth' || path === '/login') {
            switchToHeroView(false);
            showModal(elements.modalAuthRef || elements.modalAuth, false);
            return;
        }

        if (path === '/register' || path === '/signup') {
            switchToHeroView(false);
            showModal(elements.modalRegisterRef, false);
            return;
        }

        if (path === '/settings') {
            switchToHeroView(false);
            showModal(elements.modalSettings, false);
            return;
        }

        if (path === '/about') {
            switchToLandingView(false, 'about');
            return;
        }

        if (path === '/mcp') {
            switchToLandingView(false, 'mcp');
            return;
        }

        if (path === '/landing' || path === '/overview') {
            switchToLandingView(false, null);
            return;
        }

        if (path === '/star-map') {
            if (currentAssistantData) {
                openStarMap(false, false);
            } else {
                switchToHeroView(false);
            }
            return;
        }

        if (path === '/reset-password') {
            const token = new URLSearchParams(window.location.search).get('token') || '';
            switchToHeroView(false);
            if (token) {
                pendingResetToken = token;
                if (elements.refNewPassword) elements.refNewPassword.value = '';
                if (elements.refConfirmPassword) elements.refConfirmPassword.value = '';
                if (elements.refPasswordError) elements.refPasswordError.style.display = 'none';
                showModal(elements.modalChangePassword, false);
            } else {
                window.history.replaceState({}, '', '/');
                showToast(getCurrentLanguage() === 'Русский' ? 'Ссылка для сброса недействительна. Запросите новую.' : 'Reset link is invalid. Request a new one.');
                showModal(elements.modalChangePasswordEmail, false);
            }
            return;
        }

        const chatMatch = path.match(/^\/chats(?:\/([a-zA-Z0-9_\-]+))?$/);
        if (chatMatch) {
            const chatId = chatMatch[1];
            if (chatId) {
                loadChatById(chatId, false);
            } else {
                switchToHeroView(false);
            }
            return;
        }

        // Legacy /chat[/<id>] addresses redirect to canonical /chats[/<id>]
        const legacyChatMatch = path.match(/^\/chat(?:\/([a-zA-Z0-9_\-]+))?$/);
        if (legacyChatMatch) {
            const target = legacyChatMatch[1] ? `/chats/${legacyChatMatch[1]}` : '/chats';
            window.history.replaceState({ view: 'chat' }, '', target);
            renderCurrentRoute();
            return;
        }

        // Default: home / hero view
        switchToHeroView(false);
    }

    window.addEventListener('popstate', () => {
        isNavigatingFromPopState = true;
        renderCurrentRoute();
        isNavigatingFromPopState = false;
    });

    function showModal(modalEl, updateRoute = true) {
        closeSidebar();
        closeAllModals(false);
        if (modalEl) {
            modalEl.style.display = 'flex';
            if (modalEl === elements.modalSettings) {
                loadStatus();
            } else if (modalEl === elements.modalMcp) {
                updateMcpSnippets();
            }
            if (updateRoute && !isNavigatingFromPopState) {
                if (modalEl === elements.modalAuthRef || modalEl === elements.modalAuth) {
                    if (window.location.pathname !== '/login') window.history.pushState({ modal: 'login' }, '', '/login');
                } else if (modalEl === elements.modalRegisterRef) {
                    if (window.location.pathname !== '/register') window.history.pushState({ modal: 'register' }, '', '/register');
                } else if (modalEl === elements.modalSettings) {
                    if (window.location.pathname !== '/settings') window.history.pushState({ modal: 'settings' }, '', '/settings');
                } else if (modalEl === elements.modalAbout) {
                    if (window.location.pathname !== '/about') window.history.pushState({ modal: 'about' }, '', '/about');
                } else if (modalEl === elements.modalMcp) {
                    if (window.location.pathname !== '/mcp') window.history.pushState({ modal: 'mcp' }, '', '/mcp');
                }
            }
        }
    }

    function closeAllModals(restoreRoute = true) {
        document.querySelectorAll('.modal-overlay').forEach(modal => {
            modal.style.display = 'none';
        });
        if (elements.attachPopover) {
            elements.attachPopover.style.display = 'none';
        }
        if (elements.tagsPopover) {
            elements.tagsPopover.style.display = 'none';
        }
        elements.btnAttach?.classList.remove('active');
        if (restoreRoute && !isNavigatingFromPopState) {
            const path = window.location.pathname;
            if (['/login', '/auth', '/register', '/signup', '/settings', '/about', '/mcp', '/reset-password'].includes(path)) {
                window.history.pushState({}, '', '/');
            }
        }
    }

    // Modal Close buttons
    document.querySelectorAll('[data-close-modal]').forEach(btn => {
        btn.addEventListener('click', closeAllModals);
    });

    // Overlay click outside modal card closes modal
    document.querySelectorAll('.modal-overlay').forEach(overlay => {
        overlay.addEventListener('click', function (e) {
            if (e.target === overlay) {
                closeAllModals();
            }
        });
    });

    function switchToLandingView(updateRoute = true, targetAnchor = 'mcp') {
        hideStarMapImmediate();
        if (elements.heroView) elements.heroView.style.display = 'none';
        if (elements.chatView) elements.chatView.style.display = 'none';
        if (elements.profileView) elements.profileView.style.display = 'none';
        if (elements.landingView) elements.landingView.style.display = 'flex';
        if (elements.promptFooter) elements.promptFooter.classList.add('hidden');
        document.body.classList.add('landing-active');
        closeSidebar();
        closeAllModals(false);

        // Normalize anchor
        const anchorName = targetAnchor === 'landing-mcp' ? 'mcp' : 
                           (targetAnchor === 'landing-about' ? 'about' : targetAnchor);

        // Update active anchor tab in landing nav
        document.querySelectorAll('.landing-anchor-link').forEach(link => {
            const href = link.getAttribute('href');
            if (anchorName && href === `#${anchorName}`) {
                link.classList.add('active');
            } else {
                link.classList.remove('active');
            }
        });

        const targetHash = anchorName ? `#${anchorName}` : '';
        const basePath = '/landing';
        const targetUrl = `${basePath}${targetHash}`;

        if (updateRoute && !isNavigatingFromPopState) {
            if (window.location.pathname !== basePath || window.location.hash !== targetHash) {
                window.history.pushState({ view: 'landing', section: anchorName }, '', targetUrl);
            }
        }

        setTimeout(() => {
            if (!anchorName) {
                const scrollBody = document.querySelector('.landing-scroll-body');
                if (scrollBody) scrollBody.scrollTo({ top: 0, behavior: 'smooth' });
                return;
            }
            const el = document.getElementById(anchorName) || document.getElementById(`landing-${anchorName}`);
            if (el) {
                el.scrollIntoView({ behavior: 'smooth', block: 'start' });
            }
        }, 50);
    }

    // =========================================================================
    // Landing Scroll Header Logo Flying Animation
    // =========================================================================
    let isHeaderLogoVisible = false;
    let isAnimatingFly = false;

    function initLandingLogoScrollAnimation() {
        const scrollBody = document.querySelector('.landing-scroll-body');
        const headerLogo = document.getElementById('landing-header-logo');
        const welcomeBrand = document.getElementById('landing-welcome-brand');
        const langSelector = document.getElementById('landing-lang-selector');
        const navAnchorLinks = document.querySelectorAll('.landing-anchor-link');

        if (!scrollBody || !headerLogo || !welcomeBrand) return;

        function updateActiveSectionOnScroll() {
            if (!document.body.classList.contains('landing-active')) return;

            const headerRect = document.querySelector('.landing-header')?.getBoundingClientRect();
            const headerBottom = headerRect ? headerRect.bottom : 58;

            // 1. Logo & Language toggle based on hero "Errorium" position
            const brandRect = welcomeBrand.getBoundingClientRect();
            const isOutOfView = brandRect.bottom <= headerBottom + 12;

            if (isOutOfView && !isHeaderLogoVisible) {
                if (langSelector) langSelector.classList.add('hidden');
                headerLogo.classList.add('visible');
                isHeaderLogoVisible = true;
            } else if (!isOutOfView && isHeaderLogoVisible) {
                if (langSelector) langSelector.classList.remove('hidden');
                headerLogo.classList.remove('visible');
                isHeaderLogoVisible = false;
            }

            // 2. Dynamic Scrollspy: highlight anchor (#mcp or #about) when scrolled to it
            const sectionMcp = document.getElementById('mcp');
            const sectionAbout = document.getElementById('about');
            let currentActiveSection = null;

            // Trigger threshold is around header bottom + offset
            const threshold = headerBottom + 120;

            if (sectionAbout) {
                const aboutRect = sectionAbout.getBoundingClientRect();
                if (aboutRect.top <= threshold) {
                    currentActiveSection = 'about';
                }
            }

            if (!currentActiveSection && sectionMcp) {
                const mcpRect = sectionMcp.getBoundingClientRect();
                if (mcpRect.top <= threshold) {
                    currentActiveSection = 'mcp';
                }
            }

            // Update .active class on anchor tabs
            navAnchorLinks.forEach(link => {
                const href = link.getAttribute('href');
                if (currentActiveSection && href === `#${currentActiveSection}`) {
                    link.classList.add('active');
                } else {
                    link.classList.remove('active');
                }
            });
        }

        scrollBody.addEventListener('scroll', updateActiveSectionOnScroll, { passive: true });
    }

    // Modal & View Triggers
    elements.btnMcp?.addEventListener('click', (e) => {
        e.preventDefault();
        switchToLandingView(true, 'mcp');
    });

    elements.btnAbout?.addEventListener('click', (e) => {
        e.preventDefault();
        switchToLandingView(true, 'about');
    });

    elements.btnLandingBack?.addEventListener('click', () => {
        switchToHeroView(true);
    });

    elements.btnLandingCta?.addEventListener('click', () => {
        switchToHeroView(true);
        elements.promptInput?.focus();
    });

    document.getElementById('btn-landing-hero-start')?.addEventListener('click', () => {
        switchToHeroView(true);
        elements.promptInput?.focus();
    });

    document.getElementById('landing-mascot-btn')?.addEventListener('click', (e) => {
        e.preventDefault();
        switchToLandingView(true, null);
    });

    // Landing Header Language Dropdown Controller
    const btnLandingLang = document.getElementById('btn-landing-lang');
    const landingLangDropdown = document.getElementById('landing-lang-dropdown');
    btnLandingLang?.addEventListener('click', (e) => {
        e.stopPropagation();
        if (!landingLangDropdown) return;
        const isOpen = landingLangDropdown.style.display === 'flex';
        landingLangDropdown.style.display = isOpen ? 'none' : 'flex';
        btnLandingLang.setAttribute('aria-expanded', isOpen ? 'false' : 'true');
    });

    document.querySelectorAll('.landing-lang-option').forEach(option => {
        option.addEventListener('click', function (e) {
            e.stopPropagation();
            const lang = this.getAttribute('data-lang');
            if (lang) {
                const user = getAuthUser();
                if (user) {
                    user.language = lang;
                    setAuthUser(user);
                }
                applyLanguage(lang);
                showToast(lang === 'Русский' ? 'Язык переключён на Русский' : 'Language switched to English');
            }
            if (landingLangDropdown) landingLangDropdown.style.display = 'none';
            btnLandingLang?.setAttribute('aria-expanded', 'false');
        });
    });

    document.addEventListener('click', (e) => {
        if (landingLangDropdown && !btnLandingLang?.contains(e.target)) {
            landingLangDropdown.style.display = 'none';
            btnLandingLang?.setAttribute('aria-expanded', 'false');
        }
    });

    document.getElementById('landing-header-logo')?.addEventListener('click', () => {
        switchToHeroView(true);
    });

    // Landing Anchor tabs navigation
    document.querySelectorAll('.landing-anchor-link').forEach(link => {
        link.addEventListener('click', function (e) {
            e.preventDefault();
            const targetId = this.getAttribute('href')?.replace('#', '');
            if (targetId) {
                switchToLandingView(true, targetId);
            }
        });
    });

    elements.btnSettings?.addEventListener('click', () => showModal(elements.modalSettings));

    elements.btnProfile?.addEventListener('click', () => {
        if (getAuthUser()) {
            switchToProfileView();
        } else {
            showModal(elements.modalAuthRef || elements.modalAuth);
        }
    });

    // Clicking anywhere on guest footer row opens auth/login modal
    elements.sidebarUserGuest?.addEventListener('click', () => {
        showModal(elements.modalAuthRef || elements.modalAuth);
    });

    // Clicking anywhere on the authorized user footer row (name, role, gear icon or empty space) opens the profile view
    elements.sidebarUserAuth?.addEventListener('click', () => {
        switchToProfileView();
    });

    // Profile View Actions
    elements.btnProfileBack?.addEventListener('click', () => {
        switchToHeroView();
    });

    elements.profileRowSecurity?.addEventListener('click', () => {
        showModal(elements.modalResetPasswordAlert);
    });

    // =========================================================================
    // User Profile API Integration (Wave C - Task C3)
    // =========================================================================
    async function loadUserProfile() {
        if (!window.ErroriumAPI) return;
        const tokens = window.ErroriumAPI.getTokens();
        if (!tokens || !tokens.access_token) return;

        try {
            const user = await window.ErroriumAPI.request('GET', '/users/me');
            if (user) {
                setAuthUser(user);
            }
        } catch (err) {
            console.error('Failed to load user profile', err);
            const status = err && err.status;
            if (status !== 401) {
                showToast(getCurrentLanguage() === 'Русский' ? 'Ошибка загрузки профиля' : 'Failed to load profile');
            }
        }
    }

    async function patchUserProfile(patchData) {
        if (!window.ErroriumAPI) return { ok: false };
        try {
            const updated = await window.ErroriumAPI.request('PATCH', '/users/me', {
                body: patchData
            });
            if (updated) {
                setAuthUser(updated);
            }
            return { ok: true, user: updated };
        } catch (err) {
            console.error('Failed to update profile', err);
            const status = err && err.status;
            let errMsg = (err && err.error) || '';
            if (status === 409) {
                errMsg = getCurrentLanguage() === 'Русский' ? 'Email уже занят другим пользователем' : 'Email is already in use';
            } else if (status === 400) {
                errMsg = errMsg || (getCurrentLanguage() === 'Русский' ? 'Некорректные данные профиля' : 'Invalid profile data');
            } else {
                errMsg = errMsg || (getCurrentLanguage() === 'Русский' ? 'Ошибка обновления профиля' : 'Failed to update profile');
            }
            showToast(errMsg);
            return { ok: false, error: err, status: status, message: errMsg };
        }
    }

    // =========================================================================
    // Tags API & Status API (Wave C - Task C3)
    // =========================================================================
    function getTagClass(name) {
        const map = {
            'Go': 'tag-go',
            'Python': 'tag-python',
            'Java': 'tag-java',
            'C#': 'tag-csharp',
            'C++': 'tag-cpp',
            'JS': 'tag-js',
            'SQL': 'tag-sql',
            'Rust': 'tag-rust',
            'TypeScript': 'tag-ts',
            'Docker': 'tag-docker',
            'Linux': 'tag-linux',
            'Kubernetes': 'tag-k8s',
            'PostgreSQL': 'tag-postgres',
            'Redis': 'tag-redis',
            'GraphQL': 'tag-graphql',
            'React': 'tag-react',
            'Vue': 'tag-vue',
            'Swift': 'tag-swift',
            'Kotlin': 'tag-kotlin',
            'PHP': 'tag-php',
            'Git': 'tag-git',
            'Nginx': 'tag-nginx',
            'gRPC': 'tag-grpc'
        };
        return map[name] || `tag-${(name || '').toLowerCase().replace(/[^a-z0-9]/g, '')}`;
    }

    let tagsLoaded = false;
    async function loadTags() {
        if (!window.ErroriumAPI) return;
        try {
            const data = await window.ErroriumAPI.request('GET', '/tags', { auth: false });
            if (data && Array.isArray(data.items)) {
                renderTagPills(data.items);
                tagsLoaded = true;
            }
        } catch (err) {
            console.warn('Failed to load tags from API, keeping fallback pills', err);
        }
    }

    function renderTagPills(items) {
        if (!elements.tagsPopover) return;
        const grid = elements.tagsPopover.querySelector('.tags-grid');
        if (!grid) return;

        grid.innerHTML = '';
        const row = document.createElement('div');
        row.className = 'tags-row';
        row.style.flexWrap = 'wrap';
        row.style.gap = '6px';

        items.forEach(item => {
            const tagName = typeof item === 'string' ? item : item.name;
            if (!tagName) return;
            const btn = document.createElement('button');
            btn.type = 'button';
            btn.className = `tag-pill ${getTagClass(tagName)}`;
            btn.setAttribute('data-tag', tagName);
            btn.textContent = tagName;
            row.appendChild(btn);
        });
        grid.appendChild(row);
    }

    async function loadStatus() {
        if (!window.ErroriumAPI) return;
        try {
            const data = await window.ErroriumAPI.request('GET', '/status', { auth: false });
            updateStatusUI(data);
        } catch (err) {
            console.error('Failed to load status', err);
            updateStatusUI(null);
        }
    }

    function updateStatusUI(data) {
        const isRu = getCurrentLanguage() === 'Русский';
        const coreBadge = elements.settingCoreStatus || document.getElementById('setting-core-status') || document.querySelector('#modal-settings .badge');
        const isConnected = !!(data && data.search_core && data.search_core.connected);
        const addr = (data && data.search_core && data.search_core.address) ? data.search_core.address : ':50051';
        const port = addr.includes(':') ? addr.substring(addr.lastIndexOf(':')) : ':50051';

        if (coreBadge) {
            if (isConnected) {
                coreBadge.className = 'badge active-badge';
                coreBadge.textContent = isRu ? `Подключено (${port})` : `Connected (${port})`;
                coreBadge.style.color = '#10b981';
                coreBadge.style.borderColor = 'rgba(16, 185, 129, 0.3)';
                coreBadge.style.background = 'rgba(16, 185, 129, 0.1)';
            } else {
                coreBadge.className = 'badge inactive-badge';
                coreBadge.textContent = isRu ? 'Недоступно' : 'Unavailable';
                coreBadge.style.color = '#9ca3af';
                coreBadge.style.borderColor = 'rgba(156, 163, 175, 0.3)';
                coreBadge.style.background = 'rgba(156, 163, 175, 0.1)';
            }
        }

        const provider = (data && data.rag && data.rag.provider) ? data.rag.provider : 'stub';
        const ragBadge = elements.settingRagProviderBadge || document.getElementById('setting-rag-provider-badge');
        if (ragBadge) {
            ragBadge.textContent = `Provider: ${provider}`;
        }
        const ragDesc = elements.settingRagDesc || document.getElementById('setting-rag-desc');
        if (ragDesc) {
            ragDesc.textContent = isRu
                ? `Потоковая передача синтезированных ответов в реальном времени через SSE (${provider})`
                : `Stream synthesized answers in real time via SSE (${provider})`;
        }
    }

    // Profile Gender Dropdown Logic
    elements.profileInfoGenderRow?.addEventListener('click', (e) => {
        e.stopPropagation();
        if (!elements.genderDropdownPopover) return;
        const isOpen = elements.genderDropdownPopover.style.display === 'flex';
        const willOpen = !isOpen;
        // Close other dropdowns
        if (elements.roleDropdownPopover) elements.roleDropdownPopover.style.display = 'none';
        elements.profileInfoRoleRow?.classList.remove('open');
        if (elements.profileRoleChevron) elements.profileRoleChevron.style.transform = 'rotate(0deg)';
        if (elements.langDropdownPopover) elements.langDropdownPopover.style.display = 'none';
        elements.profileInfoLangRow?.classList.remove('open');
        if (elements.profileLangChevron) elements.profileLangChevron.style.transform = 'rotate(0deg)';

        elements.genderDropdownPopover.style.display = willOpen ? 'flex' : 'none';
        elements.profileInfoGenderRow.classList.toggle('open', willOpen);
        if (elements.profileGenderChevron) {
            elements.profileGenderChevron.style.transform = willOpen ? 'rotate(180deg)' : 'rotate(0deg)';
        }
    });

    document.querySelectorAll('#gender-dropdown-popover .profile-dropdown-item').forEach(item => {
        item.addEventListener('click', async function (e) {
            e.stopPropagation();
            const selectedGender = this.getAttribute('data-gender');
            if (!selectedGender) return;
            const apiGender = (selectedGender === 'Not selected' || selectedGender === 'Not specified')
                ? 'Not specified'
                : selectedGender;

            const res = await patchUserProfile({ gender: apiGender });
            if (res.ok) {
                showToast(getCurrentLanguage() === 'Русский' ? `Пол: ${selectedGender}` : `Gender set to ${selectedGender}`);
            }

            if (elements.genderDropdownPopover) elements.genderDropdownPopover.style.display = 'none';
            elements.profileInfoGenderRow?.classList.remove('open');
            if (elements.profileGenderChevron) elements.profileGenderChevron.style.transform = 'rotate(0deg)';
        });
    });

    // Profile Language Dropdown Logic
    elements.profileInfoLangRow?.addEventListener('click', (e) => {
        e.stopPropagation();
        if (!elements.langDropdownPopover) return;
        const isOpen = elements.langDropdownPopover.style.display === 'flex';
        const willOpen = !isOpen;
        // Close other dropdowns
        if (elements.roleDropdownPopover) elements.roleDropdownPopover.style.display = 'none';
        elements.profileInfoRoleRow?.classList.remove('open');
        if (elements.profileRoleChevron) elements.profileRoleChevron.style.transform = 'rotate(0deg)';
        if (elements.genderDropdownPopover) elements.genderDropdownPopover.style.display = 'none';
        elements.profileInfoGenderRow?.classList.remove('open');
        if (elements.profileGenderChevron) elements.profileGenderChevron.style.transform = 'rotate(0deg)';

        elements.langDropdownPopover.style.display = willOpen ? 'flex' : 'none';
        elements.profileInfoLangRow.classList.toggle('open', willOpen);
        if (elements.profileLangChevron) {
            elements.profileLangChevron.style.transform = willOpen ? 'rotate(180deg)' : 'rotate(0deg)';
        }
    });

    document.querySelectorAll('#lang-dropdown-popover .profile-dropdown-item').forEach(item => {
        item.addEventListener('click', async function (e) {
            e.stopPropagation();
            const selectedLang = this.getAttribute('data-lang');
            if (!selectedLang) return;
            const apiLang = (selectedLang === 'Русский' || selectedLang === 'Russian') ? 'Русский' : 'English';

            applyLanguage(apiLang);
            showToast(apiLang === 'Русский' ? 'Язык переключён на Русский' : 'Language switched to English');

            await patchUserProfile({ language: apiLang });

            if (elements.langDropdownPopover) elements.langDropdownPopover.style.display = 'none';
            elements.profileInfoLangRow?.classList.remove('open');
            if (elements.profileLangChevron) elements.profileLangChevron.style.transform = 'rotate(0deg)';
        });
    });

    elements.profileAvatarEditBadge?.addEventListener('click', (e) => {
        e.stopPropagation();
        elements.profileAvatarFileInput?.click();
    });

    elements.profileAvatarFileInput?.addEventListener('change', async (e) => {
        const file = e.target.files && e.target.files[0];
        if (!file) return;

        // Check if file is an image
        if (!file.type.startsWith('image/')) {
            showToast(getCurrentLanguage() === 'Русский' ? 'Пожалуйста, выберите файл изображения' : 'Please select a valid image file');
            return;
        }

        // Maximum size: 5 MiB
        if (file.size > 5 * 1024 * 1024) {
            showToast(getCurrentLanguage() === 'Русский' ? 'Размер файла превышает 5 МБ' : 'File size exceeds 5 MB');
            return;
        }

        const formData = new FormData();
        formData.append('avatar', file);

        try {
            const resp = await window.ErroriumAPI.request('POST', '/users/avatar', {
                body: formData,
                multipart: true,
                auth: true
            });

            const avatarUrl = resp && resp.avatar_url;
            if (avatarUrl) {
                const reader = new FileReader();
                reader.onload = function (event) {
                    const dataUrl = event.target.result;
                    extractDominantColor(dataUrl, (glowColor) => {
                        const user = getAuthUser();
                        if (user) {
                            user.avatarUrl = avatarUrl;
                            user.avatarGlow = glowColor;
                            setAuthUser(user);
                        }
                        showToast(getCurrentLanguage() === 'Русский' ? 'Аватар успешно обновлен' : 'Avatar photo and glow color updated');
                    });
                };
                reader.readAsDataURL(file);
            }
        } catch (err) {
            console.error('Failed to upload avatar', err);
            const status = err && err.status;
            let errMsg = (err && err.error) || '';
            if (status === 400) {
                errMsg = errMsg || (getCurrentLanguage() === 'Русский' ? 'Неверный формат или размер файла (400)' : 'Invalid image file or size (400)');
            } else if (status === 502) {
                errMsg = errMsg || (getCurrentLanguage() === 'Русский' ? 'Хранилище S3 недоступно (502)' : 'S3 storage unavailable (502)');
            } else {
                errMsg = errMsg || (getCurrentLanguage() === 'Русский' ? 'Ошибка загрузки аватара' : 'Failed to upload avatar');
            }
            showToast(errMsg);
        }
    });

    // Profile input change listeners (Username, Email, Role)
    elements.profileInfoUsername?.addEventListener('change', async (e) => {
        const val = e.target.value.trim();
        const user = getAuthUser();
        if (!val) {
            if (user && elements.profileInfoUsername) elements.profileInfoUsername.value = user.username;
            return;
        }
        if (user && val === user.username) return;

        const res = await patchUserProfile({ username: val });
        if (res.ok) {
            showToast(getCurrentLanguage() === 'Русский' ? 'Имя пользователя обновлено' : 'Username updated');
        } else {
            if (user && elements.profileInfoUsername) elements.profileInfoUsername.value = user.username;
        }
    });

    elements.profileInfoEmail?.addEventListener('change', async (e) => {
        const val = e.target.value.trim();
        const user = getAuthUser();
        if (!val) {
            if (user && elements.profileInfoEmail) elements.profileInfoEmail.value = user.email;
            return;
        }
        if (user && val === user.email) return;

        const res = await patchUserProfile({ email: val });
        if (res.ok) {
            showToast(getCurrentLanguage() === 'Русский' ? 'Email обновлен' : 'Email updated');
        } else {
            if (user && elements.profileInfoEmail) elements.profileInfoEmail.value = user.email;
        }
    });

    // Profile Role Dropdown Logic
    elements.profileInfoRoleRow?.addEventListener('click', (e) => {
        e.stopPropagation();
        if (!elements.roleDropdownPopover) return;
        const isOpen = elements.roleDropdownPopover.style.display === 'flex';
        const willOpen = !isOpen;
        // Close other dropdowns
        if (elements.genderDropdownPopover) elements.genderDropdownPopover.style.display = 'none';
        elements.profileInfoGenderRow?.classList.remove('open');
        if (elements.profileGenderChevron) elements.profileGenderChevron.style.transform = 'rotate(0deg)';
        if (elements.langDropdownPopover) elements.langDropdownPopover.style.display = 'none';
        elements.profileInfoLangRow?.classList.remove('open');
        if (elements.profileLangChevron) elements.profileLangChevron.style.transform = 'rotate(0deg)';

        elements.roleDropdownPopover.style.display = willOpen ? 'flex' : 'none';
        elements.profileInfoRoleRow.classList.toggle('open', willOpen);
        if (elements.profileRoleChevron) {
            elements.profileRoleChevron.style.transform = willOpen ? 'rotate(180deg)' : 'rotate(0deg)';
        }
    });

    document.querySelectorAll('.role-dropdown-item').forEach(item => {
        item.addEventListener('click', async function (e) {
            e.stopPropagation();
            const selectedRole = this.getAttribute('data-role');
            if (!selectedRole) return;

            const res = await patchUserProfile({ profession: selectedRole });
            if (res.ok) {
                showToast(getCurrentLanguage() === 'Русский' ? `Роль обновлена: ${selectedRole}` : `Role updated to ${selectedRole}`);
            }

            if (elements.roleDropdownPopover) elements.roleDropdownPopover.style.display = 'none';
            elements.profileInfoRoleRow?.classList.remove('open');
            if (elements.profileRoleChevron) elements.profileRoleChevron.style.transform = 'rotate(0deg)';
        });
    });

    document.addEventListener('click', (e) => {
        if (elements.roleDropdownPopover && !elements.profileInfoRoleRow?.contains(e.target)) {
            elements.roleDropdownPopover.style.display = 'none';
            elements.profileInfoRoleRow?.classList.remove('open');
            if (elements.profileRoleChevron) elements.profileRoleChevron.style.transform = 'rotate(0deg)';
        }
        if (elements.genderDropdownPopover && !elements.profileInfoGenderRow?.contains(e.target)) {
            elements.genderDropdownPopover.style.display = 'none';
            elements.profileInfoGenderRow?.classList.remove('open');
            if (elements.profileGenderChevron) elements.profileGenderChevron.style.transform = 'rotate(0deg)';
        }
        if (elements.langDropdownPopover && !elements.profileInfoLangRow?.contains(e.target)) {
            elements.langDropdownPopover.style.display = 'none';
            elements.profileInfoLangRow?.classList.remove('open');
            if (elements.profileLangChevron) elements.profileLangChevron.style.transform = 'rotate(0deg)';
        }
    });

    // SSO helper
    function initiateSSO(provider) {
        const prov = (provider || '').toLowerCase();
        if (prov === 'yandex') {
            if (window.ErroriumAPI && typeof window.ErroriumAPI.ssoLogin === 'function') {
                window.ErroriumAPI.ssoLogin('yandex');
            } else {
                // Redirect to /api/v1/auth/sso/login/yandex or /auth/sso/yandex/login
                window.location.href = '/api/v1/auth/sso/login/yandex';
            }
            return;
        }
        // Redirect to /auth/sso/yandex/login is active.
        // Google, GitHub, etc. remain inactive placeholders for now.
        showToast(getCurrentLanguage() === 'Русский'
            ? 'Авторизация через этот сервис находится в разработке'
            : 'Sign in via this provider is in development');
    }

    async function handleAuthCallback() {
        try {
            // 1. Check hash for tokens (redirected from gateway SSO callback)
            const hash = window.location.hash.startsWith('#') ? window.location.hash.slice(1) : window.location.hash;
            const hashParams = new URLSearchParams(hash);
            let accessToken = hashParams.get('access_token');
            let refreshToken = hashParams.get('refresh_token');

            // 2. Check query params for code and state
            if (!accessToken) {
                const searchParams = new URLSearchParams(window.location.search);
                const code = searchParams.get('code');
                const state = searchParams.get('state');
                if (code && state && window.ErroriumAPI) {
                    try {
                        const res = await window.ErroriumAPI.request('GET', `/auth/sso/callback/yandex?code=${encodeURIComponent(code)}&state=${encodeURIComponent(state)}`, { auth: false });
                        if (res && res.access_token) {
                            accessToken = res.access_token;
                            refreshToken = res.refresh_token;
                        }
                    } catch (e) {
                        console.warn('SSO callback exchange error', e);
                    }
                }
            }

            if (accessToken && window.ErroriumAPI) {
                window.ErroriumAPI.setTokens({
                    access_token: accessToken,
                    refresh_token: refreshToken || ''
                });
                try {
                    const me = await window.ErroriumAPI.request('GET', '/auth/me');
                    if (me) {
                        setAuthUser(me);
                    }
                } catch (e) {
                    console.error('Failed to load user profile after SSO', e);
                }
                showToast(getCurrentLanguage() === 'Русский' ? 'Вход через Яндекс ID выполнен успешно!' : 'Signed in with Yandex ID successfully!');
                navigateTo('/', true);
            } else {
                showToast(getCurrentLanguage() === 'Русский' ? 'Не удалось выполнить авторизацию' : 'Failed to authorize');
                navigateTo('/', true);
            }
        } catch (err) {
            console.error('SSO callback handling error', err);
            showToast(getCurrentLanguage() === 'Русский' ? 'Ошибка обработки авторизации' : 'Authorization callback error');
            navigateTo('/', true);
        }
    }

    // Logout helper
    async function handleLogout() {
        const tokens = window.ErroriumAPI ? window.ErroriumAPI.getTokens() : null;
        const refreshToken = tokens ? tokens.refresh_token : null;
        try {
            if (window.ErroriumAPI && refreshToken) {
                await window.ErroriumAPI.request('POST', '/auth/logout', {
                    body: { refresh_token: refreshToken },
                    auth: true
                });
            }
        } catch (_) {
            // Ignore API network errors during logout and finish client cleanup
        } finally {
            if (window.ErroriumAPI) {
                window.ErroriumAPI.clearTokens();
            }
            setAuthUser(null);
            closeAllModals();
            switchToHeroView();
            showToast(getCurrentLanguage() === 'Русский' ? 'Вы вышли из системы' : 'Successfully logged out');
        }
    }

    // Reference Modal Handlers
    // 1. Auth Reference Modal (Auth-screen.png)
    elements.refAuthForm?.addEventListener('submit', async (e) => {
        e.preventDefault();
        const email = elements.refAuthEmail?.value.trim() || '';
        const password = elements.refAuthPassword?.value || '';
        const rememberMe = elements.refAuthRemember ? elements.refAuthRemember.checked : true;

        if (!email || !password) {
            showToast(getCurrentLanguage() === 'Русский' ? 'Заполните email и пароль' : 'Please enter email and password');
            return;
        }

        try {
            const data = await window.ErroriumAPI.request('POST', '/auth/login', {
                body: {
                    email: email,
                    password: password,
                    remember_me: rememberMe
                },
                auth: false
            });

            if (data && data.access_token) {
                window.ErroriumAPI.setTokens({
                    access_token: data.access_token,
                    refresh_token: data.refresh_token || ''
                });
            }

            if (data && data.user) {
                setAuthUser(data.user);
            } else {
                const me = await window.ErroriumAPI.request('GET', '/auth/me');
                setAuthUser(me);
            }

            closeAllModals();
            showToast(getCurrentLanguage() === 'Русский' ? 'Успешный вход в систему!' : 'Successfully logged in!');
        } catch (err) {
            const status = err && err.status;
            let errMsg = (err && err.error) || '';
            if (status === 401) {
                errMsg = getCurrentLanguage() === 'Русский' ? 'Неверный email или пароль' : 'Invalid email or password';
                elements.refAuthEmail?.style.setProperty('border-color', '#ef4444');
                elements.refAuthPassword?.style.setProperty('border-color', '#ef4444');
            }
            showToast(errMsg || (getCurrentLanguage() === 'Русский' ? 'Ошибка входа' : 'Login failed'));
        }
    });

    elements.refBtnYandex?.addEventListener('click', () => initiateSSO('yandex'));

    elements.refAuthForgot?.addEventListener('click', () => {
        if (elements.refResetNotice) elements.refResetNotice.style.display = 'none';
        showModal(elements.modalChangePasswordEmail);
    });

    elements.refLinkSignup?.addEventListener('click', () => {
        showModal(elements.modalRegisterRef);
    });

    elements.refLinkSignin?.addEventListener('click', () => {
        showModal(elements.modalAuthRef);
    });

    elements.refRegisterForm?.addEventListener('submit', async (e) => {
        e.preventDefault();
        const username = elements.refRegisterUsername?.value.trim() || '';
        const email = elements.refRegisterEmail?.value.trim() || '';
        const password = elements.refRegisterPassword?.value || '';
        const confirm = elements.refRegisterConfirm?.value || '';
        const terms = elements.refRegisterTerms ? elements.refRegisterTerms.checked : false;

        if (!terms) {
            showToast(getCurrentLanguage() === 'Русский' ? 'Примите условия использования' : 'Please accept terms & conditions');
            return;
        }

        if (password !== confirm) {
            showToast(getCurrentLanguage() === 'Русский' ? 'Пароли не совпадают' : 'Passwords do not match');
            elements.refRegisterConfirm?.style.setProperty('border-color', '#ef4444');
            return;
        }

        if (password.length < 8) {
            showToast(getCurrentLanguage() === 'Русский' ? 'Пароль должен содержать не менее 8 символов' : 'Password must be at least 8 characters');
            elements.refRegisterPassword?.style.setProperty('border-color', '#ef4444');
            return;
        }

        try {
            const data = await window.ErroriumAPI.request('POST', '/auth/register', {
                body: {
                    username: username,
                    email: email,
                    password: password
                },
                auth: false
            });

            if (data && data.access_token) {
                window.ErroriumAPI.setTokens({
                    access_token: data.access_token,
                    refresh_token: data.refresh_token || ''
                });
            }

            if (data && data.user) {
                setAuthUser(data.user);
            } else {
                const me = await window.ErroriumAPI.request('GET', '/auth/me');
                setAuthUser(me);
            }

            closeAllModals();
            showToast(getCurrentLanguage() === 'Русский' ? 'Аккаунт успешно создан!' : 'Account created successfully!');
        } catch (err) {
            const status = err && err.status;
            let errMsg = (err && err.error) || '';
            if (status === 409) {
                errMsg = getCurrentLanguage() === 'Русский' ? 'Email уже зарегистрирован' : 'Email is already registered';
                elements.refRegisterEmail?.style.setProperty('border-color', '#ef4444');
            }
            showToast(errMsg || (getCurrentLanguage() === 'Русский' ? 'Ошибка регистрации' : 'Registration failed'));
        }
    });

    elements.refBtnRegisterYandex?.addEventListener('click', () => initiateSSO('yandex'));

    ['input', 'focus'].forEach(ev => {
        elements.refAuthEmail?.addEventListener(ev, () => {
            elements.refAuthEmail?.style.removeProperty('border-color');
        });
        elements.refAuthPassword?.addEventListener(ev, () => {
            elements.refAuthPassword?.style.removeProperty('border-color');
        });
        elements.refRegisterEmail?.addEventListener(ev, () => {
            elements.refRegisterEmail?.style.removeProperty('border-color');
        });
        elements.refRegisterPassword?.addEventListener(ev, () => {
            elements.refRegisterPassword?.style.removeProperty('border-color');
        });
        elements.refRegisterConfirm?.addEventListener(ev, () => {
            elements.refRegisterConfirm?.style.removeProperty('border-color');
        });
    });

    elements.btnSidebarLogout?.addEventListener('click', (e) => {
        e.stopPropagation();
        handleLogout();
    });

    elements.btnProfileLogout?.addEventListener('click', (e) => {
        e.stopPropagation();
        handleLogout();
    });

    elements.profileRowLogout?.addEventListener('click', (e) => {
        e.stopPropagation();
        handleLogout();
    });

    window.addEventListener('errorium:logout', () => {
        setAuthUser(null);
        updateAuthUI();
    });

    // 2. Reset Password Alert Modal (Reset-password-alert.png)
    elements.refBtnAlertSendReset?.addEventListener('click', () => {
        if (elements.refResetNotice) elements.refResetNotice.style.display = 'none';
        showModal(elements.modalChangePasswordEmail);
    });

    // 3. Change Password Email Modal (Change-password-email-screen.png)
    elements.refResetEmailForm?.addEventListener('submit', async (e) => {
        e.preventDefault();
        const isRu = getCurrentLanguage() === 'Русский';
        const email = elements.refResetEmail?.value.trim() || '';
        if (!email) {
            showToast(isRu ? 'Введите email' : 'Please enter your email');
            return;
        }

        const btn = elements.refBtnSendReset;
        if (btn) btn.disabled = true;
        if (elements.refResetNotice) elements.refResetNotice.style.display = 'none';
        try {
            await window.ErroriumAPI.request('POST', '/auth/password/forgot', {
                body: { email: email },
                auth: false
            });
            if (elements.refResetNotice) elements.refResetNotice.style.display = 'block';
            showToast(isRu ? 'Если этот email зарегистрирован, ссылка для сброса отправлена. Проверьте папку «Спам».' : 'If this email is registered, a reset link has been sent. Check your spam folder.');
        } catch (err) {
            const status = err && err.status;
            let errMsg = (err && err.error) || '';
            if (status === 400) {
                errMsg = isRu ? 'Некорректный email' : 'Invalid email format';
            } else if (status === 500) {
                errMsg = isRu ? 'Не удалось отправить письмо. Попробуйте позже.' : 'Failed to send the email. Please try again later.';
            }
            showToast(errMsg || (isRu ? 'Ошибка запроса сброса пароля' : 'Password reset request failed'));
        } finally {
            if (btn) btn.disabled = false;
        }
    });

    // 4. Change Password Modal (Change-password-screen.png) — только по ссылке из письма (/reset-password?token=...)
    elements.refChangePasswordForm?.addEventListener('submit', async (e) => {
        e.preventDefault();
        const isRu = getCurrentLanguage() === 'Русский';
        const pwd1 = elements.refNewPassword?.value || '';
        const pwd2 = elements.refConfirmPassword?.value || '';
        const showError = (msg) => {
            if (elements.refPasswordError) {
                elements.refPasswordError.textContent = msg;
                elements.refPasswordError.style.display = 'block';
            }
        };

        if (!pwd1) {
            showError(isRu ? 'Введите новый пароль' : 'Password cannot be empty');
            return;
        }
        if (pwd1.length < 8 || pwd1.length > 128) {
            showError(isRu ? 'Пароль должен содержать от 8 до 128 символов' : 'Password must be between 8 and 128 characters');
            return;
        }
        if (pwd1 !== pwd2) {
            showError(isRu ? 'Пароли не совпадают' : 'Passwords do not match');
            return;
        }
        if (!pendingResetToken) {
            showError(isRu ? 'Ссылка для сброса недействительна. Запросите новую.' : 'Reset link is invalid. Request a new one.');
            return;
        }
        if (elements.refPasswordError) {
            elements.refPasswordError.style.display = 'none';
        }

        const btn = elements.refBtnApprovePassword;
        if (btn) btn.disabled = true;
        try {
            await window.ErroriumAPI.request('POST', '/auth/password/reset', {
                body: { token: pendingResetToken, new_password: pwd1 },
                auth: false
            });
            pendingResetToken = null;
            // Бэкенд отзывает все сессии — очищаем локальное состояние тоже
            if (window.ErroriumAPI) window.ErroriumAPI.clearTokens();
            setAuthUser(null);
            window.history.replaceState({}, '', '/');
            closeAllModals(false);
            showToast(isRu ? 'Пароль успешно обновлён! Войдите с новым паролем.' : 'Password updated successfully! Log in with your new password.');
            showModal(elements.modalAuthRef || elements.modalAuth, false);
        } catch (err) {
            const status = err && err.status;
            if (status === 401) {
                showError(isRu ? 'Ссылка недействительна или истекла. Запросите новую.' : 'Reset link is invalid or expired. Request a new one.');
            } else if (status === 400) {
                showError(isRu ? 'Пароль должен содержать от 8 до 128 символов' : 'Password must be between 8 and 128 characters');
            } else {
                showError((err && err.error) || (isRu ? 'Не удалось обновить пароль' : 'Failed to update password'));
            }
        } finally {
            if (btn) btn.disabled = false;
        }
    });

    // SSO Buttons
    elements.btnLoginGithub?.addEventListener('click', () => initiateSSO('github'));
    elements.btnLoginGoogle?.addEventListener('click', () => initiateSSO('google'));

    // =========================================================================
    // MCP Integration (ADR-012: Open MCP without tokens)
    // =========================================================================
    function getMcpSseUrl() {
        const origin = (typeof window !== 'undefined' && window.location && window.location.origin)
            ? window.location.origin
            : 'https://errorium.ru';
        return origin + '/sse';
    }

    function getMcpConfigJson() {
        const sseUrl = getMcpSseUrl();
        const config = {
            mcpServers: {
                errorium: {
                    url: sseUrl
                }
            }
        };
        return JSON.stringify(config, null, 2);
    }

    function updateMcpSnippets() {
        const sseUrl = getMcpSseUrl();
        const configJson = getMcpConfigJson();
        if (elements.mcpUrlDisplay) elements.mcpUrlDisplay.value = sseUrl;
        if (elements.mcpJsonSnippet) elements.mcpJsonSnippet.textContent = configJson;
        if (elements.landingMcpUrl) elements.landingMcpUrl.textContent = sseUrl;
        if (elements.landingMcpJsonSnippet) elements.landingMcpJsonSnippet.textContent = configJson;
    }

    function openMcpModal() {
        updateMcpSnippets();
        showModal(elements.modalMcp);
    }
    window.openMcpModal = openMcpModal;

    // Copy to clipboard helpers
    const handleCopyUrl = () => {
        const url = getMcpSseUrl();
        if (navigator.clipboard && navigator.clipboard.writeText) {
            navigator.clipboard.writeText(url).then(() => {
                const isRu = getCurrentLanguage() === 'Русский';
                showToast(isRu ? 'URL подключения MCP скопирован!' : 'MCP connection URL copied to clipboard!');
            });
        }
    };

    const handleCopyMcpJson = () => {
        const json = getMcpConfigJson();
        if (navigator.clipboard && navigator.clipboard.writeText) {
            navigator.clipboard.writeText(json).then(() => {
                const isRu = getCurrentLanguage() === 'Русский';
                showToast(isRu ? 'Конфигурация MCP JSON скопирована!' : 'MCP configuration JSON copied!');
            });
        }
    };

    elements.btnCopyUrl?.addEventListener('click', handleCopyUrl);
    elements.btnCopyToken?.addEventListener('click', handleCopyUrl);
    elements.btnLandingCopyUrl?.addEventListener('click', handleCopyUrl);
    elements.btnLandingCopyToken?.addEventListener('click', handleCopyUrl);

    elements.btnCopyMcpJson?.addEventListener('click', handleCopyMcpJson);
    elements.btnLandingCopyJson?.addEventListener('click', handleCopyMcpJson);

    // Toast helper
    let toastTimeout = null;
    function showToast(message) {
        if (toastTimeout) clearTimeout(toastTimeout);
        elements.toast.textContent = message;
        elements.toast.style.display = 'block';
        toastTimeout = setTimeout(() => {
            elements.toast.style.display = 'none';
        }, 2600);
    }

    // =========================================================================
    // Auto-expanding Input & Send Logic
    // =========================================================================
    function adjustTextareaHeight() {
        if (!elements.promptInput) return;
        elements.promptInput.style.height = 'auto';
        const scrollHeight = elements.promptInput.scrollHeight;
        elements.promptInput.style.height = Math.min(scrollHeight, 140) + 'px';

        // Set rounded rectangle (border-radius: 26px) whenever there is more than 1 line
        if (elements.promptBarContainer) {
            if (scrollHeight >= 42) {
                elements.promptBarContainer.classList.add('multiline');
            } else {
                elements.promptBarContainer.classList.remove('multiline');
            }
        }
    }

    elements.promptInput?.addEventListener('input', adjustTextareaHeight);

    elements.promptInput?.addEventListener('keydown', function (e) {
        if (e.key === 'Enter' && !e.shiftKey) {
            e.preventDefault();
            submitCurrentQuery();
        }
    });

    elements.btnSend?.addEventListener('click', submitCurrentQuery);

    // Attach button popover toggle (Tags Popover - Modern Floating Dock)
    elements.btnAttach?.addEventListener('click', function (e) {
        e.stopPropagation();
        if (elements.tagsPopover) {
            const isShown = elements.tagsPopover.style.display === 'block' || elements.tagsPopover.style.display === 'flex';
            const willShow = !isShown;
            elements.tagsPopover.style.display = willShow ? 'flex' : 'none';
            elements.btnAttach.classList.toggle('active', willShow);
            if (willShow && !tagsLoaded) {
                loadTags();
            }
        }
        if (elements.attachPopover) {
            elements.attachPopover.style.display = 'none';
        }
    });

    function handleTagPillClick(tag) {
        if (!tag) return;
        if (elements.promptInput) {
            const current = elements.promptInput.value;
            elements.promptInput.value = current ? `${current} [${tag}] ` : `[${tag}] `;
        }
        if (elements.tagsPopover) {
            elements.tagsPopover.style.display = 'none';
            elements.btnAttach?.classList.remove('active');
        }
        adjustTextareaHeight();
        elements.promptInput?.focus();
    }

    // Dynamic tag clicks via event delegation on #tags-popover
    elements.tagsPopover?.addEventListener('click', function (e) {
        const pill = e.target.closest('.tag-pill');
        if (pill) {
            e.stopPropagation();
            const tag = pill.getAttribute('data-tag');
            handleTagPillClick(tag);
        }
    });

    // Tag pills click listeners (fallback/initial pills)
    document.querySelectorAll('.tag-pill').forEach(pill => {
        pill.addEventListener('click', function (e) {
            e.stopPropagation();
            const tag = this.getAttribute('data-tag');
            handleTagPillClick(tag);
        });
    });

    // Popover quick items (Preserved for compatibility)
    document.querySelectorAll('.popover-item').forEach(item => {
        item.addEventListener('click', function () {
            const action = this.getAttribute('data-action');
            if (action === 'paste-trace') {
                if (elements.promptInput) elements.promptInput.value = 'Traceback (most recent call last):\n  File "main.py", line 42, in <module>\n    process_items(data)\nTypeError: unsupported operand type(s)';
            } else if (action === 'filter-lang') {
                const lang = this.getAttribute('data-lang');
                if (elements.promptInput) elements.promptInput.value = `[${lang}] `;
            }
            if (elements.attachPopover) elements.attachPopover.style.display = 'none';
            adjustTextareaHeight();
            elements.promptInput?.focus();
        });
    });

    document.addEventListener('click', function (e) {
        if (elements.tagsPopover && !elements.tagsPopover.contains(e.target) && e.target !== elements.btnAttach && !elements.btnAttach.contains(e.target)) {
            elements.tagsPopover.style.display = 'none';
            elements.btnAttach?.classList.remove('active');
        }
        if (elements.attachPopover && !elements.attachPopover.contains(e.target) && e.target !== elements.btnAttach && !elements.btnAttach.contains(e.target)) {
            elements.attachPopover.style.display = 'none';
        }
    });

    // =========================================================================
    // Core Chat / Query Interaction
    // =========================================================================
    // Escape HTML helper (XSS protection)
    function escapeHtml(text) {
        if (text === null || text === undefined) return '';
        const div = document.createElement('div');
        div.textContent = String(text);
        return div.innerHTML;
    }

    function scrollChatToBottom() {
        if (elements.messagesContainer) {
            elements.messagesContainer.scrollTop = elements.messagesContainer.scrollHeight;
        }
    }

    function normalizeSources(rawSources) {
        if (!Array.isArray(rawSources)) return [];
        return rawSources.map(src => {
            if (!src) return { score: 0, question_score: 0, title: '', url: '#', tags: [] };
            const scoreVal = src.question_score !== undefined ? src.question_score : (src.score !== undefined ? src.score : 0);
            return {
                doc_id: src.doc_id || null,
                score: scoreVal,
                question_score: scoreVal,
                title: src.title || (src.doc_id ? `Question #${src.doc_id}` : 'Stack Overflow Source'),
                url: src.url || (src.doc_id ? `https://stackoverflow.com/questions/${src.doc_id}` : '#'),
                is_accepted: src.is_accepted !== undefined ? src.is_accepted : (src.accepted || false),
                accepted: src.is_accepted !== undefined ? src.is_accepted : (src.accepted || false),
                tags: Array.isArray(src.tags) ? src.tags : []
            };
        });
    }

    function renderAssistantCardInner(assistantData, isStreaming = false) {
        const isRu = getCurrentLanguage() === 'Русский';
        const cardTitle = isRu ? 'Errorium RAG-синтез (BM25 Ядро)' : 'Errorium RAG Synthesis (BM25 Core)';
        const solutionCodeText = isRu ? 'Код решения' : 'Solution Code';
        const copyCodeText = isRu ? 'Скопировать код' : 'Copy code';
        const starMapText = isRu ? 'Карта связей' : 'Star Map';
        const starMapTitle = isRu ? 'Открыть карту связей' : 'Open Star Map';

        const sources = assistantData.sources || [];
        const hasGraphSources = sources.length > 0 && sources.some(s => /questions\/(\d+)/.test(s.url || ''));
        const sourcesText = isRu ? `Источники Stack Overflow (${sources.length})` : `Stack Overflow Sources (${sources.length})`;
        const solution = assistantData.solution || assistantData.content || '';

        return `
            <div class="card-top-bar">
                <div class="badge-mascot">
                    <img src="/mascot.png" alt="Mascot" width="22">
                </div>
                <span class="card-title">${cardTitle}</span>
                <button class="star-map-open-btn"${hasGraphSources ? '' : ' disabled style="display: none;"'} onclick="openStarMapFromCard()" title="${starMapTitle}" aria-label="${starMapTitle}">
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
                        <circle cx="5" cy="6" r="2.2"></circle>
                        <circle cx="19" cy="5" r="2.2"></circle>
                        <circle cx="12" cy="12" r="2.6"></circle>
                        <circle cx="6" cy="19" r="2.2"></circle>
                        <circle cx="19" cy="18" r="2.2"></circle>
                        <path d="M7 7.5 10 10.5M14 10.5 17.2 6.6M10 14 7.3 17.4M14.2 13.8 17.3 16.6"></path>
                    </svg>
                    <span>${starMapText}</span>
                </button>
            </div>

            <div class="card-solution-text">${escapeHtml(solution)}${isStreaming ? '<span class="typing-cursor"></span>' : ''}</div>

            ${assistantData.code ? `
            <div class="code-block-wrapper">
                <div class="code-block-header">
                    <span>${solutionCodeText}</span>
                    <button class="code-copy-btn" onclick="copySnippet(this)">${copyCodeText}</button>
                </div>
                <pre><code>${escapeHtml(assistantData.code)}</code></pre>
            </div>
            ` : ''}

            ${(!isStreaming && sources.length > 0) ? `
            <div class="sources-card">
                <div class="sources-header" onclick="toggleSources(this)">
                    <span>${sourcesText}</span>
                    <span class="sources-arrow">▾</span>
                </div>
                <div class="sources-list">
                    ${sources.map(src => `
                        <a href="${escapeHtml(src.url)}" target="_blank" rel="noopener noreferrer" class="source-item">
                            <div class="source-left">
                                <span class="source-score">▲ ${src.score}</span>
                                <span class="source-title" title="${escapeHtml(src.title)}">${escapeHtml(src.title)}</span>
                            </div>
                            <span class="source-tags">${(src.tags || []).map(t => '#' + escapeHtml(t)).join(' ')}</span>
                        </a>
                    `).join('')}
                </div>
            </div>
            ` : ''}
        `;
    }

    function renderConversation(messagesOrUserText, assistantData = null) {
        if (!elements.messagesContainer) return;
        elements.messagesContainer.innerHTML = '';

        let messages = [];
        if (Array.isArray(messagesOrUserText)) {
            messages = messagesOrUserText;
        } else if (typeof messagesOrUserText === 'string') {
            const userText = messagesOrUserText;
            messages = [{ sender_role: 'user', content: userText }];
            if (assistantData) {
                messages.push({
                    sender_role: 'assistant',
                    content: assistantData.solution || assistantData.content || '',
                    code: assistantData.code || null,
                    rag_sources: assistantData.sources || assistantData.rag_sources || []
                });
            }
        }

        let lastAssistant = null;
        let conversationHtml = '';

        messages.forEach(msg => {
            const role = msg.sender_role || msg.senderRole || 'user';
            if (role === 'user') {
                const userText = msg.content || (msg.error_context && msg.error_context.raw_query) || '';
                currentQueryText = userText;
                conversationHtml += `
                    <!-- User Message -->
                    <div class="message-row user">
                        <div class="user-bubble">${escapeHtml(userText)}</div>
                    </div>
                `;
            } else if (role === 'assistant') {
                const solution = msg.content || '';
                const code = msg.code || null;
                const rawSources = msg.rag_sources || msg.sources || [];
                const sources = normalizeSources(rawSources);

                lastAssistant = {
                    query: currentQueryText,
                    solution: solution,
                    code: code,
                    sources: sources
                };

                conversationHtml += `
                    <!-- Assistant Solution Card -->
                    <div class="message-row assistant">
                        <div class="assistant-avatar">
                            <img src="/favicon.png" alt="Mascot" width="28" height="28">
                        </div>
                        <div class="assistant-card">
                            ${renderAssistantCardInner(lastAssistant, false)}
                        </div>
                    </div>
                `;
            }
        });

        elements.messagesContainer.innerHTML = conversationHtml;

        if (lastAssistant) {
            currentAssistantData = lastAssistant;
        }
        scrollChatToBottom();
    }

    async function loadChatById(chatId, updateRoute = true) {
        if (!chatId) return;
        currentChatId = chatId;
        switchToChatView(false);
        closeSidebar();

        if (updateRoute && !isNavigatingFromPopState) {
            const chatUrl = `/chats/${encodeURIComponent(chatId)}`;
            if (window.location.pathname !== chatUrl) {
                window.history.pushState({ view: 'chat', chatId }, '', chatUrl);
            }
        }

        updateRecentChatsActiveState(chatId);

        try {
            const res = await window.ErroriumAPI.request('GET', `/chats/${encodeURIComponent(chatId)}`);
            if (res && res.messages) {
                renderConversation(res.messages);
            } else if (res && res.chat) {
                renderConversation([]);
            }
        } catch (err) {
            showToast((err && err.error) || 'Failed to load chat');
            if (err && (err.status === 404 || err.status === 401)) {
                currentChatId = null;
                switchToHeroView();
            }
        }
    }

    function loadChatByQuery(query, chatId = null, updateRoute = true) {
        if (chatId) {
            loadChatById(chatId, updateRoute);
        } else {
            submitGuestMessage(query, query);
        }
    }

    async function submitAuthorizedMessage(query, savedQuery) {
        switchToChatView(false);

        // 1. Create chat if not present
        if (!currentChatId) {
            try {
                const newChat = await window.ErroriumAPI.request('POST', '/chats', {
                    body: { title: 'Новый чат' }
                });
                if (!newChat || !newChat.id) {
                    throw new Error('Failed to create new chat');
                }
                currentChatId = newChat.id;
                const chatUrl = `/chats/${encodeURIComponent(currentChatId)}`;
                window.history.pushState({ view: 'chat', chatId: currentChatId }, '', chatUrl);
                loadRecentChats();
            } catch (err) {
                elements.promptInput.value = savedQuery;
                adjustTextareaHeight();
                showToast((err && err.error) || 'Failed to create chat. Please try again.');
                return;
            }
        }

        let assistantRowEl = null;
        let solutionTextEl = null;
        let accumulatedText = '';

        try {
            await window.ErroriumAPI.stream(
                `/chats/${encodeURIComponent(currentChatId)}/messages`,
                { query: query },
                function (eventName, data) {
                    if (eventName === 'user_message') {
                        const userMsg = (data && data.user_message) ? data.user_message : data;
                        const userText = (userMsg && (userMsg.content || (userMsg.error_context && userMsg.error_context.raw_query))) || query;
                        currentQueryText = userText;

                        const userRow = document.createElement('div');
                        userRow.className = 'message-row user';
                        userRow.innerHTML = `<div class="user-bubble">${escapeHtml(userText)}</div>`;
                        elements.messagesContainer.appendChild(userRow);

                        assistantRowEl = document.createElement('div');
                        assistantRowEl.className = 'message-row assistant';
                        assistantRowEl.innerHTML = `
                            <div class="assistant-avatar">
                                <img src="/favicon.png" alt="Mascot" width="28" height="28">
                            </div>
                            <div class="assistant-card">
                                ${renderAssistantCardInner({ solution: '', code: null, sources: [] }, true)}
                            </div>
                        `;
                        elements.messagesContainer.appendChild(assistantRowEl);
                        solutionTextEl = assistantRowEl.querySelector('.card-solution-text');
                        scrollChatToBottom();
                    } else if (eventName === 'delta') {
                        const deltaText = (data && typeof data === 'object' && data.text !== undefined) ? data.text : (typeof data === 'string' ? data : '');
                        accumulatedText += deltaText;
                        if (solutionTextEl) {
                            solutionTextEl.textContent = accumulatedText;
                        }
                        scrollChatToBottom();
                    } else if (eventName === 'done') {
                        const assistantMsg = (data && data.assistant_message) ? data.assistant_message : null;
                        const finalContent = (assistantMsg && assistantMsg.content) || accumulatedText;
                        const finalCode = assistantMsg ? assistantMsg.code : null;
                        const rawSources = assistantMsg ? (assistantMsg.rag_sources || assistantMsg.sources || []) : [];
                        const finalSources = normalizeSources(rawSources);

                        currentAssistantData = {
                            query: query,
                            solution: finalContent,
                            code: finalCode,
                            sources: finalSources
                        };

                        if (assistantRowEl) {
                            const cardEl = assistantRowEl.querySelector('.assistant-card');
                            if (cardEl) {
                                cardEl.innerHTML = renderAssistantCardInner(currentAssistantData, false);
                            }
                        }
                        scrollChatToBottom();
                        loadRecentChats();
                    } else if (eventName === 'error') {
                        const errMsg = (data && data.error) ? data.error : 'Stream error';
                        showToast(errMsg);
                    }
                }
            );
        } catch (err) {
            showToast((err && err.error) || 'Failed to send message: network error');
            elements.promptInput.value = savedQuery;
            adjustTextareaHeight();
            if (assistantRowEl && !accumulatedText) {
                assistantRowEl.remove();
            }
        }
    }

    async function submitGuestMessage(query, savedQuery) {
        switchToChatView(false);
        // Guest queries are one-shot: every submission starts a fresh conversation
        elements.messagesContainer.innerHTML = '';
        currentAssistantData = null;
        currentQueryText = query;
        if (!isNavigatingFromPopState && window.location.pathname !== '/chats') {
            window.history.pushState({ view: 'chat' }, '', '/chats');
        }

        const userRow = document.createElement('div');
        userRow.className = 'message-row user';
        userRow.innerHTML = `<div class="user-bubble">${escapeHtml(query)}</div>`;
        elements.messagesContainer.appendChild(userRow);

        const assistantRowEl = document.createElement('div');
        assistantRowEl.className = 'message-row assistant';
        assistantRowEl.innerHTML = `
            <div class="assistant-avatar">
                <img src="/favicon.png" alt="Mascot" width="28" height="28">
            </div>
            <div class="assistant-card">
                ${renderAssistantCardInner({ solution: '', code: null, sources: [] }, true)}
            </div>
        `;
        elements.messagesContainer.appendChild(assistantRowEl);
        const solutionTextEl = assistantRowEl.querySelector('.card-solution-text');
        scrollChatToBottom();

        let accumulatedText = '';

        try {
            await window.ErroriumAPI.stream(
                '/search/rag',
                { query: query },
                function (eventName, data) {
                    if (eventName === 'delta') {
                        const deltaText = (data && typeof data === 'object' && data.text !== undefined) ? data.text : (typeof data === 'string' ? data : '');
                        accumulatedText += deltaText;
                        if (solutionTextEl) {
                            solutionTextEl.textContent = accumulatedText;
                        }
                        scrollChatToBottom();
                    } else if (eventName === 'done') {
                        const ragResp = data || {};
                        const finalContent = ragResp.synthesis || accumulatedText;
                        const finalCode = ragResp.code || null;
                        const finalSources = normalizeSources(ragResp.sources || []);

                        currentAssistantData = {
                            query: query,
                            solution: finalContent,
                            code: finalCode,
                            sources: finalSources
                        };

                        const cardEl = assistantRowEl.querySelector('.assistant-card');
                        if (cardEl) {
                            cardEl.innerHTML = renderAssistantCardInner(currentAssistantData, false);
                        }
                        scrollChatToBottom();
                    } else if (eventName === 'error') {
                        const errMsg = (data && data.error) ? data.error : 'Search error';
                        showToast(errMsg);
                    }
                },
                { auth: false }
            );
        } catch (err) {
            showToast((err && err.error) || 'Search failed. Please try again.');
            elements.promptInput.value = savedQuery;
            adjustTextareaHeight();
            if (assistantRowEl && !accumulatedText) {
                assistantRowEl.remove();
            }
        }
    }

    async function submitCurrentQuery() {
        if (!elements.promptInput) return;
        const query = elements.promptInput.value.trim();
        if (!query) return;

        const savedQuery = query;
        elements.promptInput.value = '';
        adjustTextareaHeight();

        const isAuth = !!(window.ErroriumAPI && window.ErroriumAPI.getTokens() && getAuthUser());
        if (isAuth) {
            await submitAuthorizedMessage(query, savedQuery);
        } else {
            await submitGuestMessage(query, savedQuery);
        }
    }

    function updateRecentChatsActiveState(chatId) {
        if (!elements.recentChatsList) return;
        elements.recentChatsList.querySelectorAll('.recent-chat-item').forEach(item => {
            if (chatId && item.getAttribute('data-chat-id') === chatId) {
                item.classList.add('active');
            } else {
                item.classList.remove('active');
            }
        });
    }

    async function loadRecentChats(searchQuery = '') {
        const tokens = window.ErroriumAPI ? window.ErroriumAPI.getTokens() : null;
        if (!tokens || !tokens.access_token || !getAuthUser()) {
            renderRecentChats([]);
            return;
        }

        const endpoint = '/chats?limit=50' + (searchQuery ? '&search=' + encodeURIComponent(searchQuery) : '');
        try {
            const res = await window.ErroriumAPI.request('GET', endpoint);
            chatsList = (res && Array.isArray(res.items)) ? res.items : [];
            renderRecentChats(chatsList);
        } catch (err) {
            renderRecentChats([]);
        }
    }

    function renderRecentChats(chats) {
        if (!elements.recentChatsList) return;

        const isAuth = !!(window.ErroriumAPI && window.ErroriumAPI.getTokens() && getAuthUser());

        if (!isAuth) {
            elements.recentChatsList.innerHTML = `<li class="recent-chat-empty-hint" style="padding:10px 12px;font-size:0.85rem;color:rgba(255,255,255,0.4);text-align:center;">${getCurrentLanguage() === 'Русский' ? 'Войдите, чтобы сохранять историю чатов' : 'Log in to save chat history'}</li>`;
            return;
        }

        if (!chats || chats.length === 0) {
            elements.recentChatsList.innerHTML = `<li class="recent-chat-empty-hint" style="padding:10px 12px;font-size:0.85rem;color:rgba(255,255,255,0.4);text-align:center;">${getCurrentLanguage() === 'Русский' ? 'Нет диалогов' : 'No chats yet'}</li>`;
            return;
        }

        elements.recentChatsList.innerHTML = chats.map(chat => {
            const isActive = (chat.id === currentChatId);
            const pinPrefix = chat.is_pinned ? '📌 ' : '';
            const titleEscaped = escapeHtml(chat.title || 'Новый чат');
            return `
                <li class="recent-chat-item ${isActive ? 'active' : ''}" data-chat-id="${escapeHtml(chat.id)}" data-query="${titleEscaped}">
                    <span class="chat-item-title" title="${titleEscaped}">${pinPrefix}${titleEscaped}</span>
                    <button class="chat-item-more" aria-label="More options" data-chat-id="${escapeHtml(chat.id)}">
                        <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor">
                            <circle cx="12" cy="5" r="2"></circle>
                            <circle cx="12" cy="12" r="2"></circle>
                            <circle cx="12" cy="19" r="2"></circle>
                        </svg>
                    </button>
                </li>
            `;
        }).join('');

        elements.recentChatsList.querySelectorAll('.recent-chat-item').forEach(item => {
            item.addEventListener('click', function (e) {
                if (e.target.closest('.chat-item-more')) return;
                const chatId = this.getAttribute('data-chat-id');
                loadChatById(chatId, true);
            });
        });

        elements.recentChatsList.querySelectorAll('.chat-item-more').forEach(btn => {
            btn.addEventListener('click', function (e) {
                e.stopPropagation();
                const chatId = this.getAttribute('data-chat-id');
                openChatContextMenu(chatId, this);
            });
        });
    }

    function closeChatContextMenu() {
        const existing = document.getElementById('chat-context-menu');
        if (existing) existing.remove();
    }

    document.addEventListener('click', (e) => {
        if (!e.target.closest('#chat-context-menu')) {
            closeChatContextMenu();
        }
    });

    function openChatContextMenu(chatId, buttonEl) {
        closeChatContextMenu();

        const chat = chatsList.find(c => c.id === chatId) || { id: chatId, title: '', is_pinned: false };
        const isRu = getCurrentLanguage() === 'Русский';
        const isPinned = !!chat.is_pinned;

        const menu = document.createElement('div');
        menu.className = 'chat-context-menu';
        menu.id = 'chat-context-menu';
        menu.style.position = 'fixed';
        menu.style.zIndex = '9999';
        menu.style.backgroundColor = '#161d31';
        menu.style.border = '1px solid rgba(255, 255, 255, 0.15)';
        menu.style.borderRadius = '8px';
        menu.style.padding = '4px 0';
        menu.style.minWidth = '150px';
        menu.style.boxShadow = '0 8px 24px rgba(0,0,0,0.5)';

        const renameLabel = isRu ? 'Переименовать' : 'Rename';
        const pinLabel = isPinned ? (isRu ? 'Открепить' : 'Unpin') : (isRu ? 'Закрепить' : 'Pin');
        const removeLabel = isRu ? 'Удалить' : 'Remove';

        menu.innerHTML = `
            <button class="chat-menu-action" data-action="rename" style="width:100%;text-align:left;padding:8px 12px;background:none;border:none;color:#e2e8f0;font-size:0.88rem;cursor:pointer;display:flex;align-items:center;gap:8px;">
                <span>✏️</span><span>${renameLabel}</span>
            </button>
            <button class="chat-menu-action" data-action="pin" style="width:100%;text-align:left;padding:8px 12px;background:none;border:none;color:#e2e8f0;font-size:0.88rem;cursor:pointer;display:flex;align-items:center;gap:8px;">
                <span>📌</span><span>${pinLabel}</span>
            </button>
            <div style="height:1px;background:rgba(255,255,255,0.1);margin:4px 0;"></div>
            <button class="chat-menu-action" data-action="remove" style="width:100%;text-align:left;padding:8px 12px;background:none;border:none;color:#f87171;font-size:0.88rem;cursor:pointer;display:flex;align-items:center;gap:8px;">
                <span>🗑️</span><span>${removeLabel}</span>
            </button>
        `;

        document.body.appendChild(menu);

        const rect = buttonEl.getBoundingClientRect();
        menu.style.top = (rect.bottom + 4) + 'px';
        menu.style.left = Math.max(10, rect.left - 110) + 'px';

        menu.querySelectorAll('.chat-menu-action').forEach(actionBtn => {
            actionBtn.addEventListener('mouseenter', () => {
                actionBtn.style.backgroundColor = 'rgba(255, 255, 255, 0.08)';
            });
            actionBtn.addEventListener('mouseleave', () => {
                actionBtn.style.backgroundColor = 'transparent';
            });

            actionBtn.addEventListener('click', async (e) => {
                e.stopPropagation();
                const action = actionBtn.getAttribute('data-action');
                closeChatContextMenu();

                if (action === 'rename') {
                    const newTitle = prompt(isRu ? 'Новое название чата:' : 'New chat title:', chat.title || '');
                    if (newTitle && newTitle.trim() && newTitle.trim() !== chat.title) {
                        try {
                            const updated = await window.ErroriumAPI.request('PATCH', `/chats/${encodeURIComponent(chatId)}`, {
                                body: { title: newTitle.trim() }
                            });
                            chat.title = (updated && updated.title) ? updated.title : newTitle.trim();
                            loadRecentChats();
                        } catch (err) {
                            showToast((err && err.error) || 'Failed to rename chat');
                        }
                    }
                } else if (action === 'pin') {
                    try {
                        await window.ErroriumAPI.request('PATCH', `/chats/${encodeURIComponent(chatId)}`, {
                            body: { is_pinned: !isPinned }
                        });
                        loadRecentChats();
                    } catch (err) {
                        showToast((err && err.error) || 'Failed to update pin status');
                    }
                } else if (action === 'remove') {
                    const confirmed = confirm(isRu ? 'Удалить этот чат?' : 'Delete this chat?');
                    if (confirmed) {
                        try {
                            await window.ErroriumAPI.request('DELETE', `/chats/${encodeURIComponent(chatId)}`);
                            if (currentChatId === chatId) {
                                currentChatId = null;
                                switchToHeroView();
                            }
                            loadRecentChats();
                        } catch (err) {
                            showToast((err && err.error) || 'Failed to delete chat');
                        }
                    }
                }
            });
        });
    }

    function switchToChatView(updateRoute = true) {
        hideStarMapImmediate();
        document.body.classList.remove('landing-active');
        if (elements.heroView) elements.heroView.style.display = 'none';
        if (elements.landingView) elements.landingView.style.display = 'none';
        if (elements.profileView) elements.profileView.style.display = 'none';
        if (elements.chatView) elements.chatView.style.display = 'flex';
        if (elements.promptFooter) elements.promptFooter.classList.remove('hidden');
        closeAllModals(false);
        if (updateRoute && !isNavigatingFromPopState && !window.location.pathname.startsWith('/chats')) {
            window.history.pushState({ view: 'chat' }, '', '/chats');
        }
    }

    function switchToProfileView(updateRoute = true) {
        hideStarMapImmediate();
        document.body.classList.remove('landing-active');
        if (elements.heroView) elements.heroView.style.display = 'none';
        if (elements.landingView) elements.landingView.style.display = 'none';
        if (elements.chatView) elements.chatView.style.display = 'none';
        if (elements.profileView) elements.profileView.style.display = 'flex';
        if (elements.promptFooter) elements.promptFooter.classList.add('hidden');
        closeSidebar();
        closeAllModals(false);
        loadUserProfile();
        if (updateRoute && !isNavigatingFromPopState && window.location.pathname !== '/profile') {
            window.history.pushState({ view: 'profile' }, '', '/profile');
        }
    }

    function switchToHeroView(updateRoute = true) {
        hideStarMapImmediate();
        document.body.classList.remove('landing-active');
        if (elements.profileView) elements.profileView.style.display = 'none';
        if (elements.landingView) elements.landingView.style.display = 'none';
        if (elements.chatView) elements.chatView.style.display = 'none';
        if (elements.heroView) elements.heroView.style.display = 'flex';
        if (elements.promptFooter) elements.promptFooter.classList.remove('hidden');
        if (elements.messagesContainer) elements.messagesContainer.innerHTML = '';
        currentAssistantData = null;
        currentQueryText = '';
        currentChatId = null;
        if (elements.promptInput) elements.promptInput.value = '';
        adjustTextareaHeight();
        document.querySelectorAll('.recent-chat-item').forEach(i => i.classList.remove('active'));
        closeAllModals(false);
        if (updateRoute && !isNavigatingFromPopState && window.location.pathname !== '/') {
            window.history.pushState({ view: 'home' }, '', '/');
        }
    }

    elements.btnNewChat?.addEventListener('click', () => {
        currentChatId = null;
        switchToHeroView();
        closeSidebar();
    });

    elements.headerLogo?.addEventListener('click', () => {
        currentChatId = null;
        switchToHeroView();
    });

    elements.sidebarBrand?.addEventListener('click', () => {
        switchToLandingView(true, null);
    });

    // Window helpers for dynamically rendered HTML
    window.copySnippet = function (btn) {
        const pre = btn.closest('.code-block-wrapper').querySelector('pre');
        if (pre) {
            navigator.clipboard.writeText(pre.textContent).then(() => {
                const originalText = btn.textContent;
                btn.textContent = 'Copied!';
                setTimeout(() => { btn.textContent = originalText; }, 2000);
            });
        }
    };

    window.toggleSources = function (header) {
        const list = header.nextElementSibling;
        const arrow = header.querySelector('.sources-arrow');
        if (list.style.display === 'none') {
            list.style.display = 'flex';
            arrow.textContent = '▾';
        } else {
            list.style.display = 'none';
            arrow.textContent = '▸';
        }
    };

    // =========================================================================
    // Star Map — Interactive Graph of Related SO Threads (FR-3, ADR-006)
    // Vanilla Canvas 2D + custom force simulation. Mock data until GraphService
    // integration (FR-3.5); node/edge shapes mirror protos/graph.proto.
    // =========================================================================
    let currentQueryText = '';
    let currentAssistantData = null;
    let starMapReturnUrl = '/';

    const STAR_MAP_COLORS = {
        PRIMARY_RESULT: '#10b981',
        OTHER_RESULT: '#f59e0b',
        DUPLICATE: '#38bdf8',
        LINKED: '#a855f7'
    };

    // Progressive label tiers: zoom threshold -> max title length
    const STAR_MAP_LABEL_TIERS = [
        { k: 0.7, len: 12 },
        { k: 1.15, len: 26 },
        { k: 1.8, len: 48 }
    ];

    // Edge types compatible with protos/graph.proto & openapi.yaml
    const STAR_MAP_EDGE_TYPES = {
        DUPLICATE_OF: 'DUPLICATE_OF',
        LINKED_IN_ANSWER: 'LINKED_IN_ANSWER'
    };

    function hashString(str) {
        let h = 2166136261;
        for (let i = 0; i < str.length; i++) {
            h ^= str.charCodeAt(i);
            h = Math.imul(h, 16777619);
        }
        return h >>> 0;
    }

    function mulberry32(seed) {
        let a = seed >>> 0;
        return function () {
            a |= 0;
            a = (a + 0x6D2B79F5) | 0;
            let t = Math.imul(a ^ (a >>> 15), 1 | a);
            t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
            return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
        };
    }

    function truncateText(text, limit) {
        const clean = String(text || '').replace(/\s+/g, ' ').trim();
        return clean.length > limit ? clean.slice(0, limit - 1) + '…' : clean;
    }

    function easeOutBack(x) {
        const c1 = 1.70158;
        const c3 = c1 + 1;
        return 1 + c3 * Math.pow(x - 1, 3) + c1 * Math.pow(x - 1, 2);
    }

    function easeOutCubic(x) {
        return 1 - Math.pow(1 - x, 3);
    }

    // Four-point sparkle star path centered at (0, 0), outer radius r
    function starPath(ctx, r) {
        const waist = r * 0.15;
        ctx.beginPath();
        ctx.moveTo(0, -r);
        ctx.quadraticCurveTo(waist, -waist, r, 0);
        ctx.quadraticCurveTo(waist, waist, 0, r);
        ctx.quadraticCurveTo(-waist, waist, -r, 0);
        ctx.quadraticCurveTo(-waist, -waist, 0, -r);
        ctx.closePath();
    }

    function colorWithAlpha(hex, alpha) {
        const v = parseInt(hex.slice(1), 16);
        return `rgba(${(v >> 16) & 255}, ${(v >> 8) & 255}, ${v & 255}, ${alpha})`;
    }

    // Builds a graph compatible with ThreadGraphResponse (protos/graph.proto)
    // by fetching real nodes and edges from /graph/threads via ErroriumAPI.
    async function buildStarMapGraph(assistantData) {
        const sources = (assistantData && assistantData.sources) || [];
        const docIds = [];

        sources.forEach((src) => {
            const urlMatch = /questions\/(\d+)/.exec(src.url || '');
            if (urlMatch) {
                const docId = parseInt(urlMatch[1], 10);
                if (!isNaN(docId) && !docIds.includes(docId)) {
                    docIds.push(docId);
                }
            }
        });

        if (docIds.length === 0) {
            return { nodes: [], edges: [] };
        }

        const data = await window.ErroriumAPI.request('GET', '/graph/threads?doc_ids=' + docIds.join(','));
        return {
            nodes: (data && data.nodes) || [],
            edges: (data && data.edges) || []
        };
    }
    // Canvas engine: custom force simulation (repulsion + springs + gravity + damping)
    const StarMap = {
        canvas: null, ctx: null, dpr: 1, w: 0, h: 0,
        nodes: [], edges: [], nodeById: new Map(),
        view: { x: 0, y: 0, k: 1 },
        zoomAnim: null,
        alpha: 1, raf: 0, active: false,
        hoverNode: null, selectedNode: null,
        dragNode: null, panning: false,
        pointers: new Map(), moved: 0, lastPinchDist: 0,
        stars: [], listenersBound: false, resizeObserver: null,
        reducedMotion: window.matchMedia('(prefers-reduced-motion: reduce)').matches,

        mount(graph) {
            this.canvas = elements.starMapCanvas;
            if (!this.canvas) return;
            this.ctx = this.canvas.getContext('2d');
            this.active = true;
            this.hoverNode = null;
            this.selectedNode = null;
            this.dragNode = null;
            this.panning = false;
            this.pointers.clear();
            this.zoomAnim = null;
            this.alpha = 1;

            const rand = mulberry32(hashString(currentQueryText || 'star-map'));
            const now = performance.now();
            this.nodeById = new Map();
            this.nodes = (graph.nodes || []).map((n, i) => {
                const angle = rand() * Math.PI * 2;
                const radius = n.category === 'PRIMARY_RESULT' ? 0 : 130 + rand() * 170;
                const runtime = Object.assign({}, n, {
                    doc_id: n.doc_id,
                    score: n.score || 1,
                    title: n.title || '',
                    snippet: n.snippet || '',
                    has_accepted_answer: !!n.has_accepted_answer,
                    tags: n.tags || [],
                    best_answer_score: n.best_answer_score || 0,
                    best_answer_body: n.best_answer_body || '',
                    original_url: n.original_url || '',
                    x: Math.cos(angle) * radius,
                    y: Math.sin(angle) * radius,
                    vx: 0, vy: 0,
                    r: Math.min(17, Math.max(5, 4 + Math.sqrt(Math.max(1, n.score)) * 0.15)) + (n.category === 'PRIMARY_RESULT' ? 1.5 : 0),
                    color: STAR_MAP_COLORS[n.category] || '#94a3b8',
                    tilt: (rand() - 0.5) * 0.6,
                    twinklePhase: rand() * Math.PI * 2,
                    twinkleSpeed: 0.7 + rand() * 1.1,
                    birth: this.reducedMotion ? now : now + 120 + i * 45
                });
                this.nodeById.set(n.doc_id, runtime);
                return runtime;
            });
            this.edges = (graph.edges || [])
                .map(e => ({ source: this.nodeById.get(e.source_doc_id), target: this.nodeById.get(e.target_doc_id), type: e.type }))
                .filter(e => e.source && e.target);

            this.stars = [];
            for (let i = 0; i < 140; i++) {
                this.stars.push({
                    x: Math.random(), y: Math.random(),
                    size: 0.4 + Math.random() * 1.1,
                    base: 0.12 + Math.random() * 0.4,
                    phase: Math.random() * Math.PI * 2,
                    speed: 0.4 + Math.random() * 1.2
                });
            }

            this.bindEvents();
            this.resize();
            this.fitView();
            this.hideTooltip();
            this.closeDetails(true);
            if (elements.starMapHint) elements.starMapHint.classList.remove('dismissed');
            this.canvas.style.cursor = 'grab';

            cancelAnimationFrame(this.raf);
            const loop = (t) => {
                if (!this.active) return;
                this.tick();
                this.applyViewAnimation(t);
                this.render(t);
                this.raf = requestAnimationFrame(loop);
            };
            this.raf = requestAnimationFrame(loop);
        },

        unmount() {
            this.active = false;
            cancelAnimationFrame(this.raf);
            this.nodes = [];
            this.edges = [];
            this.nodeById = new Map();
            this.hoverNode = null;
            this.selectedNode = null;
            this.dragNode = null;
            this.panning = false;
            this.pointers.clear();
            this.zoomAnim = null;
            this.hideTooltip();
            this.closeDetails(true);
        },

        resize() {
            if (!this.canvas) return;
            const rect = this.canvas.getBoundingClientRect();
            this.dpr = window.devicePixelRatio || 1;
            this.w = rect.width;
            this.h = rect.height;
            this.canvas.width = Math.max(1, Math.round(rect.width * this.dpr));
            this.canvas.height = Math.max(1, Math.round(rect.height * this.dpr));
        },

        fitTarget() {
            if (!this.nodes.length || !this.w || !this.h) return null;
            let minX = Infinity, minY = Infinity, maxX = -Infinity, maxY = -Infinity;
            this.nodes.forEach(n => {
                minX = Math.min(minX, n.x - n.r);
                maxX = Math.max(maxX, n.x + n.r);
                minY = Math.min(minY, n.y - n.r);
                maxY = Math.max(maxY, n.y + n.r);
            });
            const pad = 90;
            const k = Math.min(1.6, Math.max(0.3, Math.min(this.w / (maxX - minX + pad * 2), this.h / (maxY - minY + pad * 2))));
            return {
                k,
                x: this.w / 2 - ((minX + maxX) / 2) * k,
                y: this.h / 2 - ((minY + maxY) / 2) * k
            };
        },

        fitView() {
            const target = this.fitTarget();
            if (!target) return;
            this.view.x = target.x;
            this.view.y = target.y;
            this.view.k = target.k;
        },

        fitViewSmooth() {
            const target = this.fitTarget();
            if (target) this.animateViewTo(target, 340);
        },

        zoomAt(px, py, factor) {
            const k2 = Math.min(3.2, Math.max(0.3, this.view.k * factor));
            const ratio = k2 / this.view.k;
            this.view.x = px - (px - this.view.x) * ratio;
            this.view.y = py - (py - this.view.y) * ratio;
            this.view.k = k2;
        },

        // Smooth button zoom: repeated clicks compound onto the pending target
        zoomSmooth(px, py, factor) {
            const base = this.zoomAnim ? this.zoomAnim.to : this.view;
            const k2 = Math.min(3.2, Math.max(0.3, base.k * factor));
            const ratio = k2 / base.k;
            this.animateViewTo({
                k: k2,
                x: px - (px - base.x) * ratio,
                y: py - (py - base.y) * ratio
            }, 280);
        },

        animateViewTo(target, duration = 280) {
            if (this.reducedMotion) {
                this.zoomAnim = null;
                this.view.x = target.x;
                this.view.y = target.y;
                this.view.k = target.k;
                return;
            }
            this.zoomAnim = {
                from: { x: this.view.x, y: this.view.y, k: this.view.k },
                to: target,
                start: performance.now(),
                duration
            };
        },

        applyViewAnimation(now) {
            const anim = this.zoomAnim;
            if (!anim) return;
            const t = Math.min(1, (now - anim.start) / anim.duration);
            const e = easeOutCubic(t);
            this.view.x = anim.from.x + (anim.to.x - anim.from.x) * e;
            this.view.y = anim.from.y + (anim.to.y - anim.from.y) * e;
            this.view.k = anim.from.k + (anim.to.k - anim.from.k) * e;
            if (t >= 1) this.zoomAnim = null;
        },

        toWorld(px, py) {
            return { x: (px - this.view.x) / this.view.k, y: (py - this.view.y) / this.view.k };
        },

        pickNode(wx, wy) {
            for (let i = this.nodes.length - 1; i >= 0; i--) {
                const n = this.nodes[i];
                const dx = wx - n.x;
                const dy = wy - n.y;
                if (dx * dx + dy * dy <= (n.r + 5) * (n.r + 5)) return n;
            }
            return null;
        },

        tick() {
            if (this.alpha < 0.004) return;
            const a = this.alpha;
            const nodes = this.nodes;
            const repulsion = 2600;
            const spring = 0.028;
            const gravity = 0.012;
            const damping = 0.82;

            for (let i = 0; i < nodes.length; i++) {
                const ni = nodes[i];
                for (let j = i + 1; j < nodes.length; j++) {
                    const nj = nodes[j];
                    let dx = ni.x - nj.x;
                    let dy = ni.y - nj.y;
                    let d2 = dx * dx + dy * dy;
                    if (d2 < 1) {
                        dx = Math.random() - 0.5;
                        dy = Math.random() - 0.5;
                        d2 = dx * dx + dy * dy + 0.01;
                    }
                    if (d2 > 90000) continue;
                    const d = Math.sqrt(d2);
                    const f = (repulsion * a) / d2;
                    const fx = (dx / d) * f;
                    const fy = (dy / d) * f;
                    ni.vx += fx; ni.vy += fy;
                    nj.vx -= fx; nj.vy -= fy;
                }
            }

            this.edges.forEach(e => {
                const dx = e.target.x - e.source.x;
                const dy = e.target.y - e.source.y;
                const d = Math.sqrt(dx * dx + dy * dy) || 1;
                const rest = 105 + e.source.r + e.target.r;
                const f = (d - rest) * spring * a;
                const fx = (dx / d) * f;
                const fy = (dy / d) * f;
                e.source.vx += fx; e.source.vy += fy;
                e.target.vx -= fx; e.target.vy -= fy;
            });

            nodes.forEach(n => {
                if (n === this.dragNode) { n.vx = 0; n.vy = 0; return; }
                n.vx -= n.x * gravity * a;
                n.vy -= n.y * gravity * a;
                n.vx *= damping;
                n.vy *= damping;
                n.x += n.vx;
                n.y += n.vy;
            });

            this.alpha *= 0.982;
        },

        render(now) {
            const ctx = this.ctx;
            if (!ctx) return;
            ctx.setTransform(this.dpr, 0, 0, this.dpr, 0, 0);
            ctx.clearRect(0, 0, this.w, this.h);

            this.stars.forEach(s => {
                const twinkle = this.reducedMotion ? 1 : 0.65 + 0.35 * Math.sin(now * 0.001 * s.speed + s.phase);
                ctx.globalAlpha = s.base * twinkle;
                ctx.fillStyle = '#9db4d8';
                ctx.beginPath();
                ctx.arc(s.x * this.w, s.y * this.h, s.size, 0, Math.PI * 2);
                ctx.fill();
            });
            ctx.globalAlpha = 1;

            ctx.save();
            ctx.translate(this.view.x, this.view.y);
            ctx.scale(this.view.k, this.view.k);

            const focus = this.hoverNode || this.selectedNode;
            const neighbors = new Set();
            if (focus) {
                neighbors.add(focus.doc_id);
                this.edges.forEach(e => {
                    if (e.source === focus) neighbors.add(e.target.doc_id);
                    if (e.target === focus) neighbors.add(e.source.doc_id);
                });
            }

            this.edges.forEach(e => {
                const connected = focus && (e.source === focus || e.target === focus);
                let alpha = focus ? (connected ? 0.6 : 0.05) : 0.16;
                const birthEdge = Math.max(e.source.birth, e.target.birth);
                const edgeGrow = this.reducedMotion ? 1 : Math.min(1, Math.max(0, (now - birthEdge - 200) / 500));
                alpha *= edgeGrow;
                if (alpha <= 0.01) return;
                ctx.strokeStyle = e.type === 'DUPLICATE_OF' ? `rgba(56, 189, 248, ${alpha})` : `rgba(167, 139, 250, ${alpha})`;
                ctx.lineWidth = (connected ? 1.8 : 1.1) / this.view.k;
                ctx.setLineDash(e.type === 'DUPLICATE_OF' ? [5 / this.view.k, 5 / this.view.k] : []);
                ctx.beginPath();
                ctx.moveTo(e.source.x, e.source.y);
                ctx.lineTo(e.target.x, e.target.y);
                ctx.stroke();
            });
            ctx.setLineDash([]);

            this.nodes.forEach(n => {
                const grow = this.reducedMotion ? 1 : easeOutBack(Math.min(1, Math.max(0, (now - n.birth) / 450)));
                if (grow <= 0.01) return;
                const dimmed = focus && !neighbors.has(n.doc_id);
                const r = n.r * grow;
                const twinkle = this.reducedMotion ? 1 : 0.82 + 0.18 * Math.sin(now * 0.0016 * n.twinkleSpeed + n.twinklePhase);
                ctx.globalAlpha = dimmed ? 0.22 : 1;

                // Soft halo in the node's own color
                const haloR = r * 2.4;
                const halo = ctx.createRadialGradient(n.x, n.y, 0, n.x, n.y, haloR);
                halo.addColorStop(0, colorWithAlpha(n.color, 0.34 * twinkle));
                halo.addColorStop(0.55, colorWithAlpha(n.color, 0.1 * twinkle));
                halo.addColorStop(1, colorWithAlpha(n.color, 0));
                ctx.fillStyle = halo;
                ctx.beginPath();
                ctx.arc(n.x, n.y, haloR, 0, Math.PI * 2);
                ctx.fill();

                // Four-point sparkle body with colored glow
                ctx.save();
                ctx.translate(n.x, n.y);
                ctx.rotate(n.tilt);
                ctx.shadowColor = n.color;
                ctx.shadowBlur = (n === focus ? 24 : 14) * twinkle;
                ctx.fillStyle = n.color;
                starPath(ctx, r);
                ctx.fill();
                ctx.fill();
                ctx.shadowBlur = 0;

                // Hot white mini-star core with soft edges, aligned with the body
                const coreR = Math.max(2, r * 0.32);
                const coreGrad = ctx.createRadialGradient(0, 0, 0, 0, 0, coreR);
                coreGrad.addColorStop(0, `rgba(255, 255, 255, ${0.95 * twinkle})`);
                coreGrad.addColorStop(0.55, `rgba(255, 255, 255, ${0.55 * twinkle})`);
                coreGrad.addColorStop(1, 'rgba(255, 255, 255, 0)');
                ctx.shadowColor = `rgba(255, 255, 255, ${0.8 * twinkle})`;
                ctx.shadowBlur = 5 * twinkle;
                ctx.fillStyle = coreGrad;
                starPath(ctx, coreR);
                ctx.fill();
                ctx.shadowBlur = 0;
                ctx.restore();

                if (n === this.selectedNode) {
                    const pulse = this.reducedMotion ? 0 : Math.sin(now * 0.004) * 2;
                    ctx.strokeStyle = n.color;
                    ctx.globalAlpha = (dimmed ? 0.22 : 1) * 0.65;
                    ctx.lineWidth = 1.6 / this.view.k;
                    ctx.beginPath();
                    ctx.arc(n.x, n.y, r + 6 + pulse, 0, Math.PI * 2);
                    ctx.stroke();
                    ctx.globalAlpha = dimmed ? 0.22 : 1;
                } else if (n === this.hoverNode) {
                    ctx.strokeStyle = 'rgba(255, 255, 255, 0.75)';
                    ctx.lineWidth = 1.4 / this.view.k;
                    ctx.beginPath();
                    ctx.arc(n.x, n.y, r + 4, 0, Math.PI * 2);
                    ctx.stroke();
                }

                // Progressive labels: short at low zoom, fuller as zoom grows
                const k = this.view.k;
                const focusRelated = n === focus || (focus && neighbors.has(n.doc_id));
                let tierIdx = -1;
                for (let i = STAR_MAP_LABEL_TIERS.length - 1; i >= 0; i--) {
                    if (k >= STAR_MAP_LABEL_TIERS[i].k) { tierIdx = i; break; }
                }
                if ((tierIdx >= 0 || focusRelated) && grow > 0.6) {
                    const idx = Math.max(0, tierIdx);
                    const tier = STAR_MAP_LABEL_TIERS[idx];
                    const ramp = tierIdx < 0 ? 1 : Math.min(1, (k - tier.k) / 0.15);
                    const baseAlpha = dimmed ? 0.3 : 0.95;
                    const labelY = n.y + r + 6 / k;
                    ctx.font = `${11 / k}px "JetBrains Mono", monospace`;
                    ctx.textAlign = 'center';
                    ctx.textBaseline = 'top';
                    ctx.fillStyle = '#8fa3bf';
                    if (idx > 0 && ramp < 1) {
                        ctx.globalAlpha = baseAlpha * (1 - ramp);
                        ctx.fillText(truncateText(n.title, STAR_MAP_LABEL_TIERS[idx - 1].len), n.x, labelY);
                    }
                    ctx.globalAlpha = baseAlpha * ramp;
                    ctx.fillText(truncateText(n.title, tier.len), n.x, labelY);
                    ctx.globalAlpha = dimmed ? 0.22 : 1;
                }
            });

            ctx.globalAlpha = 1;
            ctx.restore();
        },

        showTooltip(node, px, py) {
            const tip = elements.starMapTooltip;
            if (!tip) return;
            tip.innerHTML = `
                <div class="star-map-tooltip-title">${escapeHtml(node.title)}</div>
                <div class="star-map-tooltip-snippet">${escapeHtml(node.snippet)}</div>
                <div class="star-map-tooltip-tags">${node.tags.map(tag => '#' + escapeHtml(tag)).join(' ')}</div>
            `;
            tip.style.display = 'flex';
            const tipW = tip.offsetWidth;
            const tipH = tip.offsetHeight;
            let left = px + 16;
            let top = py + 16;
            if (left + tipW > this.w - 12) left = px - tipW - 16;
            if (top + tipH > this.h - 12) top = py - tipH - 16;
            tip.style.left = Math.max(8, left) + 'px';
            tip.style.top = Math.max(8, top) + 'px';
        },

        hideTooltip() {
            if (elements.starMapTooltip) elements.starMapTooltip.style.display = 'none';
        },

        openDetails(node) {
            const panel = elements.starMapDetailPanel;
            if (!panel) return;
            const isRu = getCurrentLanguage() === 'Русский';
            const categoryNames = {
                PRIMARY_RESULT: isRu ? 'Лучший результат' : 'Top result',
                OTHER_RESULT: isRu ? 'Результат Top-K' : 'Top-K result',
                DUPLICATE: isRu ? 'Тред-дубликат' : 'Duplicate thread',
                LINKED: isRu ? 'Связанный тред' : 'Linked thread'
            };
            const acceptedText = isRu ? 'Принятый ответ' : 'Accepted answer';
            const bestText = isRu ? 'Лучший ответ' : 'Best answer';

            elements.starMapDetailCategory.textContent = categoryNames[node.category] || '';
            elements.starMapDetailCategory.style.setProperty('--cat-color', node.color);
            elements.starMapDetailTitle.textContent = node.title;
            elements.starMapDetailTags.innerHTML = (node.tags || []).map(tag => `<span class="star-map-detail-tag">#${escapeHtml(tag)}</span>`).join('');
            elements.starMapDetailAnswerLabel.textContent = node.has_accepted_answer ? acceptedText : bestText;
            elements.starMapDetailAnswer.textContent = node.best_answer_body || '';
            elements.starMapDetailLink.href = node.original_url || '#';

            panel.style.display = 'flex';
            requestAnimationFrame(() => panel.classList.add('open'));
        },

        closeDetails(immediate = false) {
            const panel = elements.starMapDetailPanel;
            if (!panel) return;
            panel.classList.remove('open');
            if (immediate) {
                panel.style.display = 'none';
            } else {
                setTimeout(() => {
                    if (!panel.classList.contains('open')) panel.style.display = 'none';
                }, 360);
            }
        },

        bindEvents() {
            if (this.listenersBound || !this.canvas) return;
            this.listenersBound = true;
            const canvas = this.canvas;

            canvas.addEventListener('pointerdown', (e) => {
                if (!this.active) return;
                this.zoomAnim = null;
                canvas.setPointerCapture(e.pointerId);
                this.pointers.set(e.pointerId, { x: e.clientX, y: e.clientY });
                if (elements.starMapHint) elements.starMapHint.classList.add('dismissed');
                if (this.pointers.size === 1) {
                    const rect = canvas.getBoundingClientRect();
                    const world = this.toWorld(e.clientX - rect.left, e.clientY - rect.top);
                    const hit = this.pickNode(world.x, world.y);
                    this.moved = 0;
                    if (hit) {
                        this.dragNode = hit;
                        this.alpha = Math.max(this.alpha, 0.4);
                        this.hideTooltip();
                    } else {
                        this.panning = true;
                    }
                    canvas.style.cursor = 'grabbing';
                } else if (this.pointers.size === 2) {
                    this.dragNode = null;
                    this.panning = false;
                    const pts = [...this.pointers.values()];
                    this.lastPinchDist = Math.hypot(pts[0].x - pts[1].x, pts[0].y - pts[1].y);
                }
            });

            canvas.addEventListener('pointermove', (e) => {
                if (!this.active) return;
                const rect = canvas.getBoundingClientRect();
                const px = e.clientX - rect.left;
                const py = e.clientY - rect.top;
                const prev = this.pointers.get(e.pointerId);

                if (prev) {
                    const dx = e.clientX - prev.x;
                    const dy = e.clientY - prev.y;
                    prev.x = e.clientX;
                    prev.y = e.clientY;
                    this.moved += Math.abs(dx) + Math.abs(dy);

                    if (this.pointers.size === 2) {
                        const pts = [...this.pointers.values()];
                        const dist = Math.hypot(pts[0].x - pts[1].x, pts[0].y - pts[1].y);
                        if (this.lastPinchDist > 0 && dist > 0) {
                            const midX = (pts[0].x + pts[1].x) / 2 - rect.left;
                            const midY = (pts[0].y + pts[1].y) / 2 - rect.top;
                            this.zoomAt(midX, midY, dist / this.lastPinchDist);
                        }
                        this.lastPinchDist = dist;
                        return;
                    }

                    if (this.dragNode) {
                        const world = this.toWorld(px, py);
                        this.dragNode.x = world.x;
                        this.dragNode.y = world.y;
                        this.dragNode.vx = 0;
                        this.dragNode.vy = 0;
                        this.alpha = Math.max(this.alpha, 0.3);
                        return;
                    }
                    if (this.panning) {
                        this.view.x += dx;
                        this.view.y += dy;
                        this.hideTooltip();
                        return;
                    }
                }

                if (this.pointers.size === 0) {
                    const world = this.toWorld(px, py);
                    const hit = this.pickNode(world.x, world.y);
                    if (hit !== this.hoverNode) {
                        this.hoverNode = hit;
                        canvas.style.cursor = hit ? 'pointer' : 'grab';
                    }
                    if (hit) this.showTooltip(hit, px, py);
                    else this.hideTooltip();
                }
            });

            const releasePointer = (e) => {
                if (!this.active) return;
                this.pointers.delete(e.pointerId);
                if (this.pointers.size < 2) this.lastPinchDist = 0;
                if (this.pointers.size > 0) return;
                const wasDrag = this.dragNode;
                const clicked = this.moved < 6;
                if (wasDrag && clicked) {
                    this.selectedNode = wasDrag;
                    this.openDetails(wasDrag);
                } else if (this.panning && clicked) {
                    this.selectedNode = null;
                    this.closeDetails();
                }
                this.dragNode = null;
                this.panning = false;
                canvas.style.cursor = this.hoverNode ? 'pointer' : 'grab';
            };
            canvas.addEventListener('pointerup', releasePointer);
            canvas.addEventListener('pointercancel', releasePointer);

            canvas.addEventListener('pointerleave', () => {
                if (!this.active || this.pointers.size > 0) return;
                this.hoverNode = null;
                this.hideTooltip();
                canvas.style.cursor = 'grab';
            });

            canvas.addEventListener('wheel', (e) => {
                if (!this.active) return;
                e.preventDefault();
                this.zoomAnim = null;
                const rect = canvas.getBoundingClientRect();
                this.zoomAt(e.clientX - rect.left, e.clientY - rect.top, Math.exp(-e.deltaY * 0.0012));
                if (elements.starMapHint) elements.starMapHint.classList.add('dismissed');
            }, { passive: false });

            if (!this.resizeObserver && window.ResizeObserver) {
                this.resizeObserver = new ResizeObserver(() => {
                    if (this.active) this.resize();
                });
                this.resizeObserver.observe(canvas);
            }
        }
    };

    // Animated view transitions: chat + prompt bar leave, Star Map appears (FR-3.4)
    async function openStarMap(animate = true, updateRoute = true) {
        if (!currentAssistantData || !elements.starMapView) return;
        if (document.body.classList.contains('star-map-active')) return;

        const isRu = getCurrentLanguage() === 'Русский';
        const sources = (currentAssistantData && currentAssistantData.sources) || [];
        const hasGraphSources = sources.length > 0 && sources.some(s => /questions\/(\d+)/.test(s.url || ''));

        if (!hasGraphSources) {
            showToast(isRu ? 'Нет источников для построения карты связей' : 'No sources available for Star Map');
            return;
        }

        let graph = null;
        try {
            graph = await buildStarMapGraph(currentAssistantData);
        } catch (err) {
            const errMsg = (err && (err.error || err.message)) || (isRu ? 'Ошибка загрузки графа связей' : 'Failed to load Star Map graph');
            showToast(errMsg);
            closeStarMap(false, updateRoute);
            return;
        }

        if (!graph || !graph.nodes || graph.nodes.length === 0) {
            showToast(isRu ? 'Граф связей пуст' : 'Star Map graph is empty');
            closeStarMap(false, updateRoute);
            return;
        }

        if (updateRoute) {
            starMapReturnUrl = window.location.pathname + window.location.search;
        }

        if (elements.starMapSubtitle) {
            elements.starMapSubtitle.textContent = truncateText(currentQueryText, 64);
        }

        if (updateRoute && !isNavigatingFromPopState && window.location.pathname !== '/star-map') {
            window.history.pushState({ view: 'star-map' }, '', '/star-map');
        }

        const finalize = () => {
            document.body.classList.remove('landing-active');
            if (elements.heroView) elements.heroView.style.display = 'none';
            if (elements.landingView) elements.landingView.style.display = 'none';
            if (elements.profileView) elements.profileView.style.display = 'none';
            if (elements.chatView) {
                elements.chatView.style.display = 'none';
                elements.chatView.classList.remove('map-leaving');
            }
            if (elements.promptFooter) {
                elements.promptFooter.classList.add('hidden');
                elements.promptFooter.classList.remove('map-leaving', 'map-entering');
            }
            elements.starMapView.style.display = 'flex';
            document.body.classList.add('star-map-active');
            if (animate) {
                elements.starMapView.classList.add('map-entering');
                setTimeout(() => elements.starMapView.classList.remove('map-entering'), 480);
            }
            StarMap.mount(graph);
        };

        if (animate && elements.chatView && elements.chatView.style.display !== 'none') {
            elements.chatView.classList.add('map-leaving');
            if (elements.promptFooter) elements.promptFooter.classList.add('map-leaving');
            setTimeout(finalize, 270);
        } else {
            finalize();
        }
    }

    function closeStarMap(animate = true, updateRoute = true) {
        if (!document.body.classList.contains('star-map-active')) return;

        const finalize = () => {
            StarMap.unmount();
            if (elements.starMapView) {
                elements.starMapView.style.display = 'none';
                elements.starMapView.classList.remove('map-leaving', 'map-entering');
            }
            document.body.classList.remove('star-map-active');
            if (elements.chatView) {
                elements.chatView.style.display = 'flex';
                if (animate) {
                    elements.chatView.classList.add('map-entering');
                    setTimeout(() => elements.chatView.classList.remove('map-entering'), 400);
                }
            }
            if (elements.promptFooter) {
                elements.promptFooter.classList.remove('hidden', 'map-leaving');
                if (animate) {
                    elements.promptFooter.classList.add('map-entering');
                    setTimeout(() => elements.promptFooter.classList.remove('map-entering'), 400);
                }
            }
            if (updateRoute && !isNavigatingFromPopState) {
                window.history.pushState({ view: 'chat' }, '', starMapReturnUrl || '/');
            }
        };

        if (animate && elements.starMapView) {
            elements.starMapView.classList.add('map-leaving');
            setTimeout(finalize, 230);
        } else {
            finalize();
        }
    }

    function hideStarMapImmediate() {
        if (!elements.starMapView) return;
        StarMap.unmount();
        elements.starMapView.style.display = 'none';
        elements.starMapView.classList.remove('map-leaving', 'map-entering');
        document.body.classList.remove('star-map-active');
        if (elements.chatView) elements.chatView.classList.remove('map-leaving', 'map-entering');
        if (elements.promptFooter) elements.promptFooter.classList.remove('map-leaving', 'map-entering');
    }

    elements.btnStarMapBack?.addEventListener('click', () => closeStarMap());
    elements.btnStarMapZoomIn?.addEventListener('click', () => StarMap.zoomSmooth(StarMap.w / 2, StarMap.h / 2, 1.25));
    elements.btnStarMapZoomOut?.addEventListener('click', () => StarMap.zoomSmooth(StarMap.w / 2, StarMap.h / 2, 0.8));
    elements.btnStarMapFit?.addEventListener('click', () => StarMap.fitViewSmooth());
    elements.btnStarMapDetailClose?.addEventListener('click', () => {
        StarMap.selectedNode = null;
        StarMap.closeDetails();
    });

    window.openStarMapFromCard = function () {
        openStarMap();
    };

    // Global keyboard shortcuts (ESC to close modals/sidebar, Ctrl/Cmd+B to toggle sidebar)
    document.addEventListener('keydown', function (e) {
        if (e.key === 'Escape') {
            if (document.body.classList.contains('star-map-active')) {
                closeStarMap();
                return;
            }
            closeAllModals();
            closeSidebar();
        } else if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'b') {
            e.preventDefault();
            if (isWideScreen()) {
                toggleDesktopSidebar();
            } else {
                elements.sidebar.classList.contains('open') ? closeSidebar() : openSidebar();
            }
        }
    });

    // =========================================================================
    // Interactive Star / Comet Cursor with Trail Effect
    // =========================================================================
    function initCursorComet() {
        if (window.matchMedia('(pointer: coarse)').matches) return;

        const canvas = document.createElement('canvas');
        canvas.className = 'cursor-comet-canvas';
        document.body.appendChild(canvas);

        const ctx = canvas.getContext('2d');
        if (!ctx) return;

        let width = canvas.width = window.innerWidth;
        let height = canvas.height = window.innerHeight;

        window.addEventListener('resize', () => {
            width = canvas.width = window.innerWidth;
            height = canvas.height = window.innerHeight;
        }, { passive: true });

        // Physics Coordinates
        let targetX = -100;
        let targetY = -100;
        let headX = -100;
        let headY = -100;
        let isFirstMove = true;
        let isRunning = false;
        let mouseInWindow = false;
        let isOverButton = false;

        // Trail history points
        const trail = [];
        const MAX_TRAIL = 16;

        function drawStar(cx, cy, spikes, outerRadius, innerRadius, color, shadowColor, shadowBlur) {
            let rot = Math.PI / 2 * 3;
            let x = cx;
            let y = cy;
            const step = Math.PI / spikes;

            ctx.save();
            ctx.beginPath();
            ctx.moveTo(cx, cy - outerRadius);
            for (let i = 0; i < spikes; i++) {
                x = cx + Math.cos(rot) * outerRadius;
                y = cy + Math.sin(rot) * outerRadius;
                ctx.lineTo(x, y);
                rot += step;

                x = cx + Math.cos(rot) * innerRadius;
                y = cy + Math.sin(rot) * innerRadius;
                ctx.lineTo(x, y);
                rot += step;
            }
            ctx.lineTo(cx, cy - outerRadius);
            ctx.closePath();

            ctx.fillStyle = color;
            if (shadowColor && shadowBlur) {
                ctx.shadowColor = shadowColor;
                ctx.shadowBlur = shadowBlur;
            }
            ctx.fill();
            ctx.restore();
        }

        function renderComet() {
            ctx.clearRect(0, 0, width, height);

            // Interpolate comet head position with smooth spring inertia
            const ease = 0.22;
            const dx = targetX - headX;
            const dy = targetY - headY;

            headX += dx * ease;
            headY += dy * ease;

            // Record trail
            if (mouseInWindow || trail.length > 0) {
                trail.unshift({ x: headX, y: headY });
                if (trail.length > MAX_TRAIL) {
                    trail.pop();
                }
            }

            // Draw comet trail (glowing ribbon of particles / segments)
            for (let i = trail.length - 1; i >= 1; i--) {
                const p1 = trail[i];
                const p0 = trail[i - 1];
                const ratio = 1 - (i / trail.length);
                const size = ratio * 6.5;

                ctx.save();
                ctx.beginPath();
                ctx.moveTo(p1.x, p1.y);
                ctx.lineTo(p0.x, p0.y);
                ctx.lineWidth = size;
                ctx.lineCap = 'round';
                ctx.strokeStyle = `rgba(56, 189, 248, ${ratio * 0.45})`;
                ctx.shadowColor = '#38bdf8';
                ctx.shadowBlur = ratio * 14;
                ctx.stroke();
                ctx.restore();

                // Sparkle dust particle along trail
                if (i % 3 === 0) {
                    ctx.save();
                    ctx.beginPath();
                    ctx.arc(p1.x, p1.y, ratio * 2.2, 0, Math.PI * 2);
                    ctx.fillStyle = `rgba(147, 197, 253, ${ratio * 0.7})`;
                    ctx.shadowColor = '#93c5fd';
                    ctx.shadowBlur = 8;
                    ctx.fill();
                    ctx.restore();
                }
            }

            // Draw leading comet / star head only when not hovering interactive buttons
            if (mouseInWindow && !isOverButton && headX > 0 && headY > 0) {
                // Soft radial gradient glow around the star (no sharp circle stroke/fill)
                const glowGrad = ctx.createRadialGradient(headX, headY, 0, headX, headY, 28);
                glowGrad.addColorStop(0, 'rgba(56, 189, 248, 0.45)');
                glowGrad.addColorStop(0.35, 'rgba(56, 189, 248, 0.18)');
                glowGrad.addColorStop(0.7, 'rgba(37, 99, 235, 0.06)');
                glowGrad.addColorStop(1, 'rgba(56, 189, 248, 0)');

                ctx.save();
                ctx.beginPath();
                ctx.arc(headX, headY, 28, 0, Math.PI * 2);
                ctx.fillStyle = glowGrad;
                ctx.fill();
                ctx.restore();

                // 4-pointed glowing Star head
                drawStar(headX, headY, 4, 8, 3, '#ffffff', '#38bdf8', 16);

                // Pure white hot center
                ctx.save();
                ctx.beginPath();
                ctx.arc(headX, headY, 2, 0, Math.PI * 2);
                ctx.fillStyle = '#ffffff';
                ctx.shadowColor = '#ffffff';
                ctx.shadowBlur = 10;
                ctx.fill();
                ctx.restore();
            }

            // Loop lifecycle
            const dist = Math.hypot(dx, dy);
            if (dist > 0.1 || trail.length > 0 || mouseInWindow) {
                requestAnimationFrame(renderComet);
            } else {
                isRunning = false;
            }
        }

        window.addEventListener('mousemove', (e) => {
            targetX = e.clientX;
            targetY = e.clientY;
            mouseInWindow = true;

            // Check if cursor is directly hovering over a button
            const targetEl = e.target;
            const buttonEl = targetEl && (targetEl.closest('button, [role="button"], input[type="button"], input[type="submit"]') !== null);
            isOverButton = buttonEl;

            if (isFirstMove) {
                headX = targetX;
                headY = targetY;
                isFirstMove = false;
            }

            if (!canvas.classList.contains('active')) {
                canvas.classList.add('active');
            }

            if (!isRunning) {
                isRunning = true;
                requestAnimationFrame(renderComet);
            }
        }, { passive: true });

        document.addEventListener('mouseleave', () => {
            mouseInWindow = false;
            canvas.classList.remove('active');
        });

        document.addEventListener('mouseenter', () => {
            mouseInWindow = true;
            canvas.classList.add('active');
        });
    }

    // Initialize authentication UI state from stored session
    updateAuthUI();

    // Initialize landing scroll animation
    initLandingLogoScrollAnimation();

    // Initialize cursor comet / star trail
    initCursorComet();

    // Preload dynamic tags catalog from API
    loadTags();

    // Initialize SPA route based on initial URL
    renderCurrentRoute();

})();
