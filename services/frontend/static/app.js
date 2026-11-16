/**
 * Errorium Single Page Application (Pure Vanilla JS, Zero Frameworks)
 * Implements chat, search, drawer navigation, MCP modal, and responsive interactions
 */

(function () {
    'use strict';

    // =========================================================================
    // Preset Knowledge Base for Default Recent Chats
    // =========================================================================
    const KNOWLEDGE_BASE = {
        'chat-1': {
            title: 'SyntaxError: invalid syntax',
            query: 'SyntaxError: invalid syntax',
            solution: 'This error occurs when Python parser encounters code that violates language grammar rules. The most common causes are missing colons after control flow statements (`if`, `for`, `def`), unclosed parentheses or quotes, or using reserved keywords as identifiers.',
            code: '# ❌ Incorrect:\ndef calculate_total(items)\n    total = 0\n    return total\n\n# ✅ Fixed (add colon and check matching braces):\ndef calculate_total(items):\n    total = 0\n    return total',
            sources: [
                {
                    title: 'SyntaxError: invalid syntax in Python (Top causes & fixes)',
                    score: 4180,
                    accepted: true,
                    tags: ['python', 'syntax-error', 'python-3.x'],
                    url: 'https://stackoverflow.com/questions/27539308'
                },
                {
                    title: 'Why am I getting SyntaxError on the line after my print statement?',
                    score: 1240,
                    accepted: true,
                    tags: ['python', 'print-statement'],
                    url: 'https://stackoverflow.com/questions/15478423'
                }
            ]
        },
        'chat-2': {
            title: 'IndentationError: unexpected indent',
            query: 'IndentationError: unexpected indent',
            solution: 'Python uses indentation to define code blocks. An `unexpected indent` occurs when a line has more leading whitespace or tabs than expected, or when spaces and tabs are mixed within the same file.',
            code: '# ❌ Incorrect:\ndef process_data():\n    step1 = True\n      step2 = False  # Unexpected indentation level\n\n# ✅ Fixed (use consistent 4 spaces per block):\ndef process_data():\n    step1 = True\n    step2 = False',
            sources: [
                {
                    title: 'How to fix Python IndentationError: unexpected indent',
                    score: 2890,
                    accepted: true,
                    tags: ['python', 'indentation', 'pep8'],
                    url: 'https://stackoverflow.com/questions/11256501'
                }
            ]
        },
        'chat-3': {
            title: 'TypeError: ‘list’ object cannot be interpre...',
            query: "TypeError: 'list' object cannot be interpreted as an integer",
            solution: 'This error happens when passing a list to a function or slice index that expects an integer (e.g. `range(my_list)` instead of `range(len(my_list))` or passing an array into `int()` conversion).',
            code: '# ❌ Incorrect:\nitems = [10, 20, 30]\nfor i in range(items):  # Passing list instead of length\n    print(i)\n\n# ✅ Fixed:\nfor i in range(len(items)):\n    print(items[i])\n\n# Or idiomatically:\nfor item in items:\n    print(item)',
            sources: [
                {
                    title: 'TypeError: \'list\' object cannot be interpreted as an integer in range()',
                    score: 1950,
                    accepted: true,
                    tags: ['python', 'list', 'typeerror'],
                    url: 'https://stackoverflow.com/questions/35593121'
                }
            ]
        },
        'chat-4': {
            title: 'NameError: name ‘foobar’ is not defined',
            query: "NameError: name 'foobar' is not defined",
            solution: 'A variable, function, or module name was referenced before it was declared, imported, or within an inaccessible scope.',
            code: '# ❌ Incorrect:\nprint(foobar)\nfoobar = "active"\n\n# ✅ Fixed:\nfoobar = "active"\nprint(foobar)',
            sources: [
                {
                    title: 'NameError: name is not defined in Python',
                    score: 3410,
                    accepted: true,
                    tags: ['python', 'scope', 'name-error'],
                    url: 'https://stackoverflow.com/questions/14872225'
                }
            ]
        },
        'chat-5': {
            title: 'UnboundLocalError: local variable ‘foobar’...',
            query: "UnboundLocalError: local variable 'foobar' referenced before assignment",
            solution: 'Occurs inside a function when assigning to a variable that has the same name as an outer/global variable, making Python treat it as local throughout the function before its assignment statement is reached.',
            code: 'foobar = 10\n\ndef increment():\n    # ❌ foobar += 1  # Raises UnboundLocalError\n    global foobar     # ✅ Explicitly declare global or return new value\n    foobar += 1\n\nincrement()',
            sources: [
                {
                    title: 'UnboundLocalError: local variable referenced before assignment',
                    score: 5120,
                    accepted: true,
                    tags: ['python', 'variable-assignment', 'scope'],
                    url: 'https://stackoverflow.com/questions/10851906'
                }
            ]
        },
        'chat-6': {
            title: 'TypeError: ‘str’ object does not support it...',
            query: "TypeError: 'str' object does not support item assignment",
            solution: 'Strings in Python are immutable objects. You cannot modify an individual character in-place using indexing (`s[0] = "x"`). You must construct a new string or convert to a mutable sequence like a list.',
            code: 'text = "hello"\n\n# ❌ text[0] = "H"  # TypeError\n\n# ✅ Fixed (slice and concatenate):\ntext = "H" + text[1:]\n\n# Or using list conversion:\nchars = list(text)\nchars[0] = "H"\ntext = "".join(chars)',
            sources: [
                {
                    title: 'How to replace a character in a Python string by index?',
                    score: 3740,
                    accepted: true,
                    tags: ['python', 'string', 'immutability'],
                    url: 'https://stackoverflow.com/questions/4175294'
                }
            ]
        },
        'chat-7': {
            title: 'KeyError: ‘spam’',
            query: "KeyError: 'spam'",
            solution: 'Raised when trying to access a dictionary key that does not exist. Use the `.get(key, default)` method or `in` operator to safely verify membership before reading.',
            code: 'data = {"name": "Alice"}\n\n# ❌ value = data["spam"]  # Raises KeyError\n\n# ✅ Fixed:\nvalue = data.get("spam", "default_val")\n\n# Or with membership check:\nif "spam" in data:\n    print(data["spam"])',
            sources: [
                {
                    title: 'How to handle KeyError exceptions when accessing Python dictionaries',
                    score: 2210,
                    accepted: true,
                    tags: ['python', 'dictionary', 'keyerror'],
                    url: 'https://stackoverflow.com/questions/11041405'
                }
            ]
        }
    };

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
            "Встречайте новые функции!": "Meet new features!",
            "Бэкенд-разработчик": "Backend developer",
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
            "Синхронизируйте историю ошибок, знания команды и создавайте токены MCP для Cursor и Claude Code.": "Sync your recent error history, team knowledge, and generate MCP API tokens for Cursor & Claude Code.",
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
            "Мы отправили вам письмо.": "We've sent you an email.",
            "Не забудьте проверить папку «Спам»": "Don't forget to check your spam folder",
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
            "Ранжированные источники Stack Overflow": "Ranked Stack Overflow Sources",
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
            "Errorium строго следует академическим ограничениям: полный запрет готовых поисковых систем (Elasticsearch, Solr, Lucene, rank-bm25). Инвертированный индекс, списки постингов, токенизация кода и ранжирование BM25 полностью написаны с нуля и непрерывно верифицируются метриками Recall@k и MRR.": "Errorium enforces strict academic guidelines: zero reliance on ready-made search engines (Elasticsearch, Solr, Lucene, rank-bm25). The inverted index, memory-mapped postings list, code tokenization, and BM25 ranking formulas are completely custom-built and continuously measured against Recall@k and MRR benchmarks."
        },
        ru: {
            "Recent chats": "Недавние чаты",
            "New chat": "Новый чат",
            "About us": "О нас",
            "Log in": "Войти",
            "Meet new features!": "Встречайте новые функции!",
            "Backend developer": "Бэкенд-разработчик",
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
            "Sync your recent error history, team knowledge, and generate MCP API tokens for Cursor & Claude Code.": "Синхронизируйте историю ошибок, знания команды и создавайте токены MCP для Cursor и Claude Code.",
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
            "We've sent you an email.": "Мы отправили вам письмо.",
            "Don't forget to check your spam folder": "Не забудьте проверить папку «Спам»",
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
            "Ranked Stack Overflow Sources": "Ранжированные источники Stack Overflow",
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
            "Errorium enforces strict academic guidelines: zero reliance on ready-made search engines (Elasticsearch, Solr, Lucene, rank-bm25). The inverted index, memory-mapped postings list, code tokenization, and BM25 ranking formulas are completely custom-built and continuously measured against Recall@k and MRR benchmarks.": "Errorium строго следует академическим ограничениям: полный запрет готовых поисковых систем (Elasticsearch, Solr, Lucene, rank-bm25). Инвертированный индекс, списки постингов, токенизация кода и ранжирование BM25 полностью написаны с нуля и непрерывно верифицируются метриками Recall@k и MRR."
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

        // 6. Update user role in sidebar if default
        if (elements.sidebarAuthRole && (elements.sidebarAuthRole.textContent === 'Backend developer' || elements.sidebarAuthRole.textContent === 'Бэкенд-разработчик')) {
            elements.sidebarAuthRole.textContent = isRu ? 'Бэкенд-разработчик' : 'Backend developer';
        }
        if (elements.profileInfoRole && (elements.profileInfoRole.value === 'Backend developer' || elements.profileInfoRole.value === 'Бэкенд-разработчик')) {
            elements.profileInfoRole.value = isRu ? 'Бэкенд-разработчик' : 'Backend developer';
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

        btnCopyToken: document.getElementById('btn-copy-token'),
        btnCopyMcpJson: document.getElementById('btn-copy-mcp-json'),
        mcpTokenDisplay: document.getElementById('mcp-token-display'),
        mcpJsonSnippet: document.getElementById('mcp-json-snippet'),

        btnLandingCopyToken: document.getElementById('btn-landing-copy-token'),
        btnLandingCopyJson: document.getElementById('btn-landing-copy-json'),
        landingMcpToken: document.getElementById('landing-mcp-token'),
        landingMcpJsonSnippet: document.getElementById('landing-mcp-json-snippet'),

        btnLoginGithub: document.getElementById('btn-login-github'),
        btnLoginGoogle: document.getElementById('btn-login-google'),

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

    // Recent chats search filter
    elements.searchRecentChats?.addEventListener('input', function (e) {
        const query = e.target.value.toLowerCase().trim();
        const items = elements.recentChatsList ? elements.recentChatsList.querySelectorAll('.recent-chat-item') : [];
        items.forEach(item => {
            const title = item.querySelector('.chat-item-title').textContent.toLowerCase();
            if (title.includes(query)) {
                item.style.display = 'flex';
            } else {
                item.style.display = 'none';
            }
        });
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
                localStorage.setItem(AUTH_USER_KEY, JSON.stringify(user));
            } else {
                localStorage.removeItem(AUTH_USER_KEY);
            }
        } catch (e) {}
        updateAuthUI();
    }

    function updateAuthUI() {
        const user = getAuthUser();
        if (user) {
            if (elements.sidebarUserGuest) elements.sidebarUserGuest.style.display = 'none';
            if (elements.sidebarUserAuth) elements.sidebarUserAuth.style.display = 'flex';
            if (elements.headerAvatarGuest) elements.headerAvatarGuest.style.display = 'none';
            if (elements.headerAvatarAuth) {
                elements.headerAvatarAuth.style.display = 'flex';
                if (user.avatarUrl) {
                    elements.headerAvatarAuth.innerHTML = `<img src="${user.avatarUrl}" style="width:100%;height:100%;border-radius:50%;object-fit:cover;" alt="Avatar" />`;
                } else {
                    elements.headerAvatarAuth.textContent = (user.avatarLetter || user.username[0] || 'T').toUpperCase();
                }
            }

            if (elements.sidebarAuthAvatar) {
                if (user.avatarUrl) {
                    elements.sidebarAuthAvatar.innerHTML = `<img src="${user.avatarUrl}" style="width:100%;height:100%;border-radius:50%;object-fit:cover;" alt="Avatar" />`;
                } else {
                    elements.sidebarAuthAvatar.textContent = (user.avatarLetter || user.username[0] || 'T').toUpperCase();
                }
            }
            if (elements.sidebarAuthName) elements.sidebarAuthName.textContent = user.username;
            if (elements.sidebarAuthRole) elements.sidebarAuthRole.textContent = user.role || 'Backend developer';

            if (elements.profileAvatarLetter) {
                elements.profileAvatarLetter.textContent = (user.avatarLetter || user.username[0] || 'T').toUpperCase();
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
                elements.profileInfoEmail.value = user.email || 'Test_user@mail.ru';
                elements.profileInfoEmail.textContent = user.email || 'Test_user@mail.ru';
            }
            if (elements.profileInfoRole) {
                elements.profileInfoRole.textContent = user.role || 'Backend developer';
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

        const chatMatch = path.match(/^\/chat(?:\/([a-zA-Z0-9_\-]+))?$/);
        if (chatMatch) {
            const chatId = chatMatch[1];
            if (chatId && KNOWLEDGE_BASE[chatId]) {
                loadChatByQuery(KNOWLEDGE_BASE[chatId].query || KNOWLEDGE_BASE[chatId].title, chatId, false);
            } else if (chatId) {
                loadChatByQuery(decodeURIComponent(chatId.replace(/-/g, ' ')), chatId, false);
            } else {
                switchToChatView(false);
            }
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
            if (['/login', '/auth', '/register', '/signup', '/settings', '/about', '/mcp'].includes(path)) {
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
        item.addEventListener('click', function (e) {
            e.stopPropagation();
            const selectedGender = this.getAttribute('data-gender');
            if (!selectedGender) return;
            const user = getAuthUser() || {
                username: 'Test_user',
                email: 'Test_user@mail.ru',
                role: 'Backend developer',
                gender: 'Male',
                language: 'English',
                avatarLetter: 'T'
            };
            user.gender = selectedGender;
            setAuthUser(user);
            showToast(`Gender set to ${selectedGender}`);

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
        item.addEventListener('click', function (e) {
            e.stopPropagation();
            const selectedLang = this.getAttribute('data-lang');
            if (!selectedLang) return;
            const user = getAuthUser();
            if (user) {
                user.language = selectedLang;
                setAuthUser(user);
            }
            applyLanguage(selectedLang);
            showToast(selectedLang === 'Русский' ? 'Язык переключён на Русский' : 'Language switched to English');

            if (elements.langDropdownPopover) elements.langDropdownPopover.style.display = 'none';
            elements.profileInfoLangRow?.classList.remove('open');
            if (elements.profileLangChevron) elements.profileLangChevron.style.transform = 'rotate(0deg)';
        });
    });

    elements.profileAvatarEditBadge?.addEventListener('click', (e) => {
        e.stopPropagation();
        elements.profileAvatarFileInput?.click();
    });

    elements.profileAvatarFileInput?.addEventListener('change', (e) => {
        const file = e.target.files && e.target.files[0];
        if (!file) return;

        // Check if file is an image
        if (!file.type.startsWith('image/')) {
            showToast('Please select a valid image file');
            return;
        }

        const reader = new FileReader();
        reader.onload = function (event) {
            const dataUrl = event.target.result;
            extractDominantColor(dataUrl, (glowColor) => {
                const user = getAuthUser() || {
                    username: 'Test_user',
                    email: 'Test_user@mail.ru',
                    role: 'Backend developer',
                    gender: 'Male',
                    language: 'English',
                    avatarLetter: 'T'
                };
                user.avatarUrl = dataUrl;
                user.avatarGlow = glowColor;
                setAuthUser(user);
                showToast('Avatar photo and glow color updated');
            });
        };
        reader.readAsDataURL(file);
    });

    // Profile input change listeners (Username, Email, Role)
    elements.profileInfoUsername?.addEventListener('change', (e) => {
        const val = e.target.value.trim();
        if (!val) return;
        const user = getAuthUser() || {
            username: 'Test_user',
            email: 'Test_user@mail.ru',
            role: 'Backend developer',
            gender: 'Male',
            language: 'English',
            avatarLetter: 'T'
        };
        user.username = val;
        user.avatarLetter = val[0].toUpperCase();
        setAuthUser(user);
        showToast('Username updated');
    });

    elements.profileInfoEmail?.addEventListener('change', (e) => {
        const val = e.target.value.trim();
        if (!val) return;
        const user = getAuthUser() || {
            username: 'Test_user',
            email: 'Test_user@mail.ru',
            role: 'Backend developer',
            gender: 'Male',
            language: 'English',
            avatarLetter: 'T'
        };
        user.email = val;
        setAuthUser(user);
        showToast('Email updated');
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
        item.addEventListener('click', function (e) {
            e.stopPropagation();
            const selectedRole = this.getAttribute('data-role');
            if (!selectedRole) return;
            const user = getAuthUser() || {
                username: 'Test_user',
                email: 'Test_user@mail.ru',
                role: 'Backend developer',
                gender: 'Male',
                language: 'English',
                avatarLetter: 'T'
            };
            user.role = selectedRole;
            setAuthUser(user);
            showToast(`Role updated to ${selectedRole}`);

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

    // Reference Modal Handlers
    // 1. Auth Reference Modal (Auth-screen.png)
    elements.refAuthForm?.addEventListener('submit', (e) => {
        e.preventDefault();
        const email = elements.refAuthEmail?.value || 'Test_user@mail.ru';
        const user = {
            username: 'Test_user',
            email: email,
            role: 'Backend developer',
            gender: 'Male',
            language: 'English',
            avatarLetter: 'T'
        };
        setAuthUser(user);
        closeAllModals();
        showToast('Successfully logged in!');
    });

    elements.refBtnYandex?.addEventListener('click', () => {
        const user = {
            username: 'Test_user',
            email: 'Test_user@mail.ru',
            role: 'Backend developer',
            gender: 'Male',
            language: 'English',
            avatarLetter: 'T'
        };
        setAuthUser(user);
        closeAllModals();
        showToast('Logged in via Yandex ID');
    });

    elements.refAuthForgot?.addEventListener('click', () => {
        showModal(elements.modalChangePasswordEmail);
    });

    elements.refLinkSignup?.addEventListener('click', () => {
        showModal(elements.modalRegisterRef);
    });

    elements.refLinkSignin?.addEventListener('click', () => {
        showModal(elements.modalAuthRef);
    });

    elements.refRegisterForm?.addEventListener('submit', (e) => {
        e.preventDefault();
        const username = elements.refRegisterUsername?.value.trim() || 'New_user';
        const email = elements.refRegisterEmail?.value.trim() || 'user@example.com';
        const password = elements.refRegisterPassword?.value || '';
        const confirm = elements.refRegisterConfirm?.value || '';

        if (password !== confirm) {
            showToast('Passwords do not match');
            return;
        }

        const user = {
            username: username,
            email: email,
            role: 'Backend developer',
            gender: 'Male',
            language: 'English',
            avatarLetter: (username[0] || 'U').toUpperCase()
        };
        setAuthUser(user);
        closeAllModals();
        showToast('Account created successfully!');
    });

    elements.refBtnRegisterYandex?.addEventListener('click', () => {
        const user = {
            username: 'Yandex_user',
            email: 'yandex_user@yandex.ru',
            role: 'Backend developer',
            gender: 'Male',
            language: 'English',
            avatarLetter: 'Y'
        };
        setAuthUser(user);
        closeAllModals();
        showToast('Signed up via Yandex ID');
    });

    // 2. Reset Password Alert Modal (Reset-password-alert.png)
    elements.refBtnAlertSendReset?.addEventListener('click', () => {
        showModal(elements.modalChangePasswordEmail);
    });

    // 3. Change Password Email Modal (Change-password-email-screen.png)
    elements.refResetEmailForm?.addEventListener('submit', (e) => {
        e.preventDefault();
        if (elements.refResetNotice) {
            elements.refResetNotice.style.display = 'block';
        }
        showToast(getCurrentLanguage() === 'Русский' ? 'Письмо для сброса отправлено! Проверьте папку «Спам».' : 'Reset email sent! Check your spam folder.');
        setTimeout(() => {
            showModal(elements.modalChangePassword);
            if (elements.refResetNotice) {
                elements.refResetNotice.style.display = 'none';
            }
        }, 1600);
    });

    // 4. Change Password Modal (Change-password-screen.png)
    elements.refChangePasswordForm?.addEventListener('submit', (e) => {
        e.preventDefault();
        const pwd1 = elements.refNewPassword?.value || '';
        const pwd2 = elements.refConfirmPassword?.value || '';
        if (!pwd1) {
            if (elements.refPasswordError) {
                elements.refPasswordError.textContent = 'Password cannot be empty';
                elements.refPasswordError.style.display = 'block';
            }
            return;
        }
        if (pwd1 !== pwd2) {
            if (elements.refPasswordError) {
                elements.refPasswordError.textContent = 'Passwords do not match';
                elements.refPasswordError.style.display = 'block';
            }
            return;
        }
        if (elements.refPasswordError) {
            elements.refPasswordError.style.display = 'none';
        }
        closeAllModals();
        showToast('Password updated successfully!');
    });

    // Fast Auth Demo
    elements.btnLoginGithub?.addEventListener('click', () => handleLogin('GitHub'));
    elements.btnLoginGoogle?.addEventListener('click', () => handleLogin('Google'));

    function handleLogin(provider) {
        const user = {
            username: 'Test_user',
            email: 'dev@errorium.ru',
            role: 'Backend developer',
            gender: 'Male',
            language: 'English',
            avatarLetter: 'T'
        };
        setAuthUser(user);
        closeAllModals();
        showToast(`Signed in via ${provider} SSO! MCP tokens active.`);
    }

    // Copy to clipboard helpers
    elements.btnCopyToken?.addEventListener('click', () => {
        if (elements.mcpTokenDisplay) {
            navigator.clipboard.writeText(elements.mcpTokenDisplay.value).then(() => {
                showToast('Bearer token copied to clipboard!');
            });
        }
    });

    elements.btnCopyMcpJson?.addEventListener('click', () => {
        if (elements.mcpJsonSnippet) {
            navigator.clipboard.writeText(elements.mcpJsonSnippet.textContent).then(() => {
                showToast('MCP configuration JSON copied!');
            });
        }
    });

    // Landing Copy to clipboard helpers
    elements.btnLandingCopyToken?.addEventListener('click', () => {
        if (elements.landingMcpToken) {
            navigator.clipboard.writeText(elements.landingMcpToken.textContent.trim()).then(() => {
                showToast('Bearer token copied to clipboard!');
            });
        }
    });

    elements.btnLandingCopyJson?.addEventListener('click', () => {
        if (elements.landingMcpJsonSnippet) {
            navigator.clipboard.writeText(elements.landingMcpJsonSnippet.textContent).then(() => {
                showToast('MCP configuration JSON copied!');
            });
        }
    });

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
        }
        if (elements.attachPopover) {
            elements.attachPopover.style.display = 'none';
        }
    });

    // Tag pills click listeners
    document.querySelectorAll('.tag-pill').forEach(pill => {
        pill.addEventListener('click', function (e) {
            e.stopPropagation();
            const tag = this.getAttribute('data-tag');
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
    function submitCurrentQuery() {
        if (!elements.promptInput) return;
        const query = elements.promptInput.value.trim();
        if (!query) return;

        elements.promptInput.value = '';
        adjustTextareaHeight();
        loadChatByQuery(query);
    }

    function switchToChatView(updateRoute = true) {
        document.body.classList.remove('landing-active');
        if (elements.heroView) elements.heroView.style.display = 'none';
        if (elements.landingView) elements.landingView.style.display = 'none';
        if (elements.profileView) elements.profileView.style.display = 'none';
        if (elements.chatView) elements.chatView.style.display = 'flex';
        if (elements.promptFooter) elements.promptFooter.classList.remove('hidden');
        closeAllModals(false);
        if (updateRoute && !isNavigatingFromPopState && !window.location.pathname.startsWith('/chat')) {
            window.history.pushState({ view: 'chat' }, '', '/chat');
        }
    }

    function switchToProfileView(updateRoute = true) {
        document.body.classList.remove('landing-active');
        if (elements.heroView) elements.heroView.style.display = 'none';
        if (elements.landingView) elements.landingView.style.display = 'none';
        if (elements.chatView) elements.chatView.style.display = 'none';
        if (elements.profileView) elements.profileView.style.display = 'flex';
        if (elements.promptFooter) elements.promptFooter.classList.add('hidden');
        closeSidebar();
        closeAllModals(false);
        if (updateRoute && !isNavigatingFromPopState && window.location.pathname !== '/profile') {
            window.history.pushState({ view: 'profile' }, '', '/profile');
        }
    }

    function switchToHeroView(updateRoute = true) {
        document.body.classList.remove('landing-active');
        if (elements.profileView) elements.profileView.style.display = 'none';
        if (elements.landingView) elements.landingView.style.display = 'none';
        if (elements.chatView) elements.chatView.style.display = 'none';
        if (elements.heroView) elements.heroView.style.display = 'flex';
        if (elements.promptFooter) elements.promptFooter.classList.remove('hidden');
        if (elements.messagesContainer) elements.messagesContainer.innerHTML = '';
        if (elements.promptInput) elements.promptInput.value = '';
        adjustTextareaHeight();
        document.querySelectorAll('.recent-chat-item').forEach(i => i.classList.remove('active'));
        closeAllModals(false);
        if (updateRoute && !isNavigatingFromPopState && window.location.pathname !== '/') {
            window.history.pushState({ view: 'home' }, '', '/');
        }
    }

    elements.btnNewChat?.addEventListener('click', () => {
        switchToHeroView();
        closeSidebar();
    });

    elements.headerLogo?.addEventListener('click', () => {
        switchToHeroView();
    });

    elements.sidebarBrand?.addEventListener('click', () => {
        switchToLandingView(true, null);
    });

    // Render Chat Messages
    function loadChatByQuery(query, chatId = null, updateRoute = true) {
        switchToChatView(false);
        closeSidebar();

        if (updateRoute && !isNavigatingFromPopState) {
            const chatUrl = chatId ? `/chat/${encodeURIComponent(chatId)}` : '/chat';
            if (window.location.pathname !== chatUrl) {
                window.history.pushState({ view: 'chat', chatId, query }, '', chatUrl);
            }
        }

        // Mark active in sidebar if known
        document.querySelectorAll('.recent-chat-item').forEach(item => {
            if (chatId && item.getAttribute('data-chat-id') === chatId) {
                item.classList.add('active');
            } else {
                item.classList.remove('active');
            }
        });

        // Resolve data from KB or synthesize on-the-fly
        let data = chatId ? KNOWLEDGE_BASE[chatId] : null;
        if (!data) {
            // Find match in KB by fuzzy title
            for (const key in KNOWLEDGE_BASE) {
                if (KNOWLEDGE_BASE[key].title.toLowerCase().includes(query.toLowerCase()) ||
                    query.toLowerCase().includes(KNOWLEDGE_BASE[key].title.toLowerCase())) {
                    data = KNOWLEDGE_BASE[key];
                    break;
                }
            }
        }

        if (!data) {
            data = {
                title: query,
                query: query,
                solution: `Analyzed error with BM25 inverted index ranking ($k_1=1.5, b=0.75$). The exception indicates an invalid runtime operation or unresolved reference in your execution environment.`,
                code: `// Diagnostic verification:\ntry {\n    // Execute target operation\n    verifyInputParameters();\n} catch (Exception ex) {\n    System.err.println("Resolved: " + ex.getMessage());\n}`,
                sources: [
                    {
                        title: `Stack Overflow: Solutions and discussions for "${escapeHtml(query)}"`,
                        score: 1840,
                        accepted: true,
                        tags: ['error', 'debugging', 'stack-trace'],
                        url: 'https://stackoverflow.com/search?q=' + encodeURIComponent(query)
                    }
                ]
            };
        }

        renderConversation(query, data);
    }

    function renderConversation(userText, assistantData) {
        const isRu = getCurrentLanguage() === 'Русский';
        const cardTitle = isRu ? 'Errorium RAG-синтез (BM25 Ядро)' : 'Errorium RAG Synthesis (BM25 Core)';
        const solutionCodeText = isRu ? 'Код решения' : 'Solution Code';
        const copyCodeText = isRu ? 'Скопировать код' : 'Copy code';
        const sourcesText = isRu ? `Ранжированные источники Stack Overflow (${assistantData.sources.length})` : `Ranked Stack Overflow Sources (${assistantData.sources.length})`;

        elements.messagesContainer.innerHTML = `
            <!-- User Message -->
            <div class="message-row user">
                <div class="user-bubble">${escapeHtml(userText)}</div>
            </div>

            <!-- Assistant Solution Card -->
            <div class="message-row assistant">
                <div class="assistant-card">
                    <div class="card-top-bar">
                        <div class="badge-mascot">
                            <img src="/mascot.png" alt="Mascot" width="22">
                        </div>
                        <span class="card-title">${cardTitle}</span>
                    </div>

                    <div class="card-solution-text">
                        ${assistantData.solution}
                    </div>

                    ${assistantData.code ? `
                    <div class="code-block-wrapper">
                        <div class="code-block-header">
                            <span>${solutionCodeText}</span>
                            <button class="code-copy-btn" onclick="copySnippet(this)">${copyCodeText}</button>
                        </div>
                        <pre><code>${escapeHtml(assistantData.code)}</code></pre>
                    </div>
                    ` : ''}

                    <div class="sources-card">
                        <div class="sources-header" onclick="toggleSources(this)">
                            <span>${sourcesText}</span>
                            <span class="sources-arrow">▾</span>
                        </div>
                        <div class="sources-list">
                            ${assistantData.sources.map(src => `
                                <a href="${src.url}" target="_blank" rel="noopener noreferrer" class="source-item">
                                    <div class="source-left">
                                        <span class="source-score">▲ ${src.score}</span>
                                        <span class="source-title" title="${src.title}">${src.title}</span>
                                    </div>
                                    <span class="source-tags">${src.tags.map(t => '#' + t).join(' ')}</span>
                                </a>
                            `).join('')}
                        </div>
                    </div>
                </div>
            </div>
        `;
    }

    // Attach click listeners to default recent chat items
    document.querySelectorAll('.recent-chat-item').forEach(item => {
        item.addEventListener('click', function (e) {
            // Ignore clicking 3 dots button
            if (e.target.closest('.chat-item-more')) {
                e.stopPropagation();
                showToast('Chat options: Rename / Pin / Remove');
                return;
            }
            const chatId = this.getAttribute('data-chat-id');
            const query = this.getAttribute('data-query');
            loadChatByQuery(query, chatId);
        });
    });

    // Escape HTML helper
    function escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

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

    // Global keyboard shortcuts (ESC to close modals/sidebar, Ctrl/Cmd+B to toggle sidebar)
    document.addEventListener('keydown', function (e) {
        if (e.key === 'Escape') {
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

    // Initialize SPA route based on initial URL
    renderCurrentRoute();

})();
