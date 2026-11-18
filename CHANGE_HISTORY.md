# CHANGE_HISTORY.md — История изменений проекта Errorium

## [2026-11-18] — Язык индексатора (Python) и спецификация токенизатора (Роль 1)

### Затронутые компоненты и спецификации:
- `docs/Tech/03-data.md` (новый раздел 5: токенизация и подготовка текста)
- `docs/Service/decisions.md` (ADR-008)
- `services/search_engine_indexer/README.md`
- `STATUS.md`

### Суть изменений:
1. **ADR-008:** индексатор и токенизатор — Python 3 + стандартная библиотека; потоковый построчный разбор `<row .../>`, чтение архива подпроцессом `7zz x -so`.
2. **Спецификация токенизатора (версия `"1"`):** очистка HTML, неделимые токены (`NullPointerException`, `C++`, `C#`, `.NET`, `fmt.Println`, `os.path.join`, `pip3`, `docker-compose`), подтокены по границам camelCase/`_`/`.`/`-` как дополнение к целому токену, без стемминга и стоп-слов (обоснование — п. 5.3), определение $|d|$ и $TF$, таблица эталонных примеров для unit-тестов, контракт с Ролью 2 (общий модуль токенизатора, `tokenizer_version` в `metadata.json`).
3. Согласован с командой состав индексируемого текста: `Title` + `Body` вопроса + `Body` принятого ответа (Роль 2 подстраивается).

### Результаты тестов/бенчмарка:
- Код не менялся, тесты и бенчмарк не запускались.

## [2026-11-18] — Спецификация потокового чтения дампа SO и хеш-выборки корпуса (Роль 1)

### Затронутые компоненты и спецификации:
- `docs/Tech/03-data.md` (новый раздел 4)
- `docs/Tech/02-running.md`
- `docs/Service/decisions.md` (ADR-007)
- `services/search_engine_indexer/README.md`
- `STATUS.md`

### Суть изменений:
1. **Замеры исходного архива** `stackoverflow.com.7z` (68 946 936 349 байт): архив несолидный, `Posts.xml` — 105.6 ГБ (24.14 млн вопросов, 36.08 млн ответов), `PostLinks.xml` — 0.8 ГБ, `Tags.xml` — 5.9 МБ. `Posts.xml` не отсортирован по `Id`, вопросы и ответы перемешаны; на префиксе 3 ГиБ найден только 12% принятых ответов.
2. **Решение (ADR-007):** сырые XML не распаковываются; поток `7zz x -so`; корпус формируется детерминированной хеш-выборкой тредов (ответ — по хешу `ParentId`) с расширением соседями из `PostLinks.xml`, иначе граф тредов вырождается ($p^2$). Префиксный срез запрещён.
3. **Режимы:** `sample` (КТ2, ≈ 1% тредов, ≈ 12–15 мин, ≈ 2–3 ГиБ диска), `full` (КТ3, шардирование по хешу + слияние блоков SPIMI).
4. **Переменные окружения** индексатора (`SO_DUMP_ARCHIVE`, `INDEX_MODE`, `SAMPLE_*`, `GRAPH_MAX_NEIGHBORS`, `SHARD_COUNT`), критерии приёмки (RAM ≤ 4 ГБ, детерминизм, `sample_manifest.json`) и план unit-тестов `tests/unit/test_sampler.py`.

### Результаты тестов/бенчмарка:
- Код не менялся, тесты и бенчмарк не запускались. Числовые оценки размера выборки помечены в спецификации как подлежащие замеру.

## [2026-11-16] — Star Map: мягкие края ядра и удаление рейтинга/галочки со всех поверхностей карты (Milestone 1, Iteration 93)

### Затронутые компоненты и спецификации:
- `docs/Product/03-requirements.md` — FR-3.3: hover без числового рейтинга и индикатора принятого ответа.
- `docs/Product/04-interface-and-behavior.md` — «Экран 2.1: Star Map»: ядро с мягкими размытыми краями; подписи без строки метаданных; тултип и панель деталей без рейтинга и галочки.
- `services/frontend/static/app.js` — ядро-мини-звезда рендерится с мягкими краями (radial gradient с растворением к кончикам лучей + белое свечение shadowBlur); удалена строка метаданных `▲ score · ✅` из прогрессивных подписей; удалены мета-строки из hover-тултипа и панели деталей (`starMapDetailMeta` убран из кэша элементов).
- `services/frontend/static/index.html` — удалён контейнер `#star-map-detail-meta`.
- `services/frontend/static/styles.css` — удалены мёртвые стили `.star-map-tooltip-meta/-accepted/-score` и `.star-map-detail-meta/-score/-accepted`.
- `tests/unit/test_star_map.py` — проверки мягкого ядра (`coreGrad`, gradient stops); отсутствие рейтинга/галочки проверяется в секции Star Map (✅ в мок-коде чата KNOWLEDGE_BASE не относится к карте).
- `STATUS.md`, `CHANGE_HISTORY.md`.

### Суть изменений:
1. **Мягкое ядро:** белая мини-звезда больше не имеет резких краёв — заливка radial gradient (0.95 → 0.55 → 0 к кончикам лучей) плюс белое свечение (shadowBlur 5, мерцает вместе с ядром). Форма соосна телу (поворот 45° удалён в Iteration 92).
2. **Рейтинг и галочка убраны** со всех поверхностей карты: подписи узлов (ступени 12 → 26 → 48 символов, без мета-строки), hover-тултип (заголовок + сниппет + теги), панель деталей (категория + заголовок + теги + ответ + ссылка). Поля `score` / `has_accepted_answer` / `best_answer_score` сохранены в данных (контракт graph.proto) и по-прежнему влияют на размер узла и заголовок секции ответа («Принятый ответ» / «Лучший ответ»).

### Верификация:
- `node --check services/frontend/static/app.js` — синтаксис валиден.
- `python3 -m unittest discover -s tests/unit -p "test_*.py"` — **38/38 тестов OK**.


## [2026-11-16] — Star Map: белая мини-звезда ядра соосна телу (убран поворот на 45°) (Milestone 1, Iteration 92)

### Затронутые компоненты и спецификации:
- `docs/Product/04-interface-and-behavior.md` — «Экран 2.1: Star Map»: ядро описано как соосное телу, без относительного поворота.
- `services/frontend/static/app.js` — удалён `ctx.rotate(Math.PI / 4)` перед рендером ядра: внутренняя белая мини-звезда теперь повторяет ориентацию внешней (общий детерминированный наклон узла сохраняется).
- `tests/unit/test_star_map.py` — в `test_star_map_progressive_labels_and_compact_stars` проверка поворота заменена на `assertNotIn("ctx.rotate(Math.PI / 4)")`.
- `CHANGE_HISTORY.md`.

### Суть изменений:
1. Поворот ядра на 45° (эксперимент с «восьмилучевым бликом» из Iteration 91) не понравился пользователю — лучи внутренней и внешней звёзд теперь направлены одинаково, форма читается как единая звезда с ярким центром.

### Верификация:
- `node --check services/frontend/static/app.js` — синтаксис валиден.
- `python3 -m unittest discover -s tests/unit -p "test_*.py"` — **38/38 тестов OK**.


## [2026-11-16] — Star Map: компактные звёзды с белой мини-звездой внутри и прогрессивные подписи по зуму (Milestone 1, Iteration 91)

### Затронутые компоненты и спецификации:
- `docs/Product/04-interface-and-behavior.md` — «Экран 2.1: Star Map»: уточнена форма узлов (компактный радиус $4 + 0.15\sqrt{Score}$, clamp $[5,17]$, $+1.5$ для Top-1; ядро — белая мини-звезда, повёрнутая на 45°) и добавлены прогрессивные подписи (ступени 0.7/1.15/1.8 с crossfade и строкой метаданных).
- `services/frontend/static/app.js` — радиус узла уменьшен (~65% от прежнего); белое ядро-кружок заменено на мини-sparkle (`starPath`, поворот `Math.PI / 4`); константа `STAR_MAP_LABEL_TIERS` и прогрессивный рендер подписей: сокращённые (~12 символов) → средние (~26) → полные (~48) + строка `▲ score · ✅`, плавный crossfade между ступенями (ramp 0.15 по k); наведённый/выбранный узел и соседи подписаны на любом зуме.
- `tests/unit/test_star_map.py` — новый тест `test_star_map_progressive_labels_and_compact_stars`.
- `STATUS.md`, `CHANGE_HISTORY.md`.

### Суть изменений:
1. **Звёзды поменьше:** радиус `4 + 0.15·√score` (clamp [5, 17], +1.5 для Top-1) вместо `6 + 0.22·√score` (clamp [7, 26], +2) — например, score 4180: 15.2 вместо 22.2; ореол (2.4r) уменьшился пропорционально. Зона клика (r + 5) и физика не изменились.
2. **Белая мини-звезда внутри:** вместо белого круга ядро рендерится как sparkle радиусом `max(2, 0.32r)`, повёрнутый на 45° относительно тела — получается восьмилучевой блик; мерцание ядра сохранено.
3. **Прогрессивные подписи:** при zoom ≥ 0.7 подписи появляются сокращёнными (12 символов), на ≥ 1.15 удлиняются до 26, на ≥ 1.8 — до 48 символов плюс строка метаданных (▲ score · ✅). Переходы между ступенями — crossfade (старая длина гаснет, новая проявляется на окне Δk = 0.15), без скачков. Focus-узлы и их соседи подписаны всегда (минимум 12 символов).

### Верификация:
- `node --check services/frontend/static/app.js` — синтаксис валиден.
- `python3 -m unittest discover -s tests/unit -p "test_*.py"` — **38/38 тестов OK** (добавлен `test_star_map_progressive_labels_and_compact_stars`).
- Node smoke-тест: выбор ступени и ramp на k = 0.5/0.7/0.85/1.15/1.225/1.3/1.8/2.0/3.2, монотонный рост информативности (12 → 26 → 48 + meta), длины truncateText, компактные радиусы и клампы — OK.


## [2026-11-16] — Star Map: узлы-звёзды со свечением своего цвета вместо разноцветных кружков (Milestone 1, Iteration 90)

### Затронутые компоненты и спецификации:
- `docs/Product/04-interface-and-behavior.md` — «Экран 2.1: Star Map», добавлено описание формы узлов (четырёхлучевые sparkle-звёзды с горячим ядром, цветным ореолом и мерцанием; звёздчатые маркеры легенды).
- `services/frontend/static/app.js` — хелперы `starPath` (sparkle-путь на quadratic-кривых) и `colorWithAlpha` (hex → rgba); узлы получают детерминированные `tilt`, `twinklePhase`, `twinkleSpeed`; рендер узла заменён с залитого круга на: мягкий ореол своего цвета (radial gradient) → четырёхлучевая звезда с цветным glow (двойной fill для усиления свечения) → горячее белое ядро; лёгкое мерцание ореола/glow/ядра (отключается при `prefers-reduced-motion`).
- `services/frontend/static/styles.css` — маркеры легенды `.star-map-dot` стали звёздочками (clip-path polygon + drop-shadow своего цвета вместо круглых точек с box-shadow).
- `services/frontend/static/index.html` — версия кэша styles.css поднята до `v=20261116_23`.
- `tests/unit/test_star_map.py` — новый тест `test_star_map_star_shaped_nodes`.
- `STATUS.md`, `CHANGE_HISTORY.md`.

### Суть изменений:
1. Узлы карты теперь выглядят как маленькие звёзды созвездия, а не разноцветные кружки: четырёхлучевая sparkle-форма с вогнутой талией (waist = 0.15r), горячим белым ядром (~0.16r) и мягким ореолом радиусом 2.4r в цвете категории (FR-3.2 цвета и размер ∝ sqrt(score) сохранены).
2. Каждая звезда имеет небольшой детерминированный наклон (±0.3 рад) и индивидуальное мерцание (амплитуда 18%, скорость 0.7–1.8) — карта «дышит» как ночное небо; при reduced motion мерцание и наклон-анимация отключаются.
3. Hover/selection кольца, подсветка связей, затемнение не-соседей, каскадное «рождение» и физика не изменились; зона клика осталась круглой (r + 5) для удобства попадания.
4. Легенда повторяет форму узлов: звёздочки на CSS clip-path со свечением drop-shadow своего цвета.

### Верификация:
- `node --check services/frontend/static/app.js` — синтаксис валиден.
- `python3 -m unittest discover -s tests/unit -p "test_*.py"` — **37/37 тестов OK** (добавлен `test_star_map_star_shaped_nodes`).
- Node smoke-тест формы: sparkle-путь (4 луча, 4 quadratic-кривые, вогнутая талия, замкнутость) и конвертация hex → rgba для всех четырёх цветов категорий — OK.


## [2026-11-16] — Star Map: плавный анимированный зум по кнопкам +/−/⤢ (Milestone 1, Iteration 89)

### Затронутые компоненты и спецификации:
- `docs/Product/04-interface-and-behavior.md` — «Экран 2.1: Star Map», уточнено поведение кнопок масштаба (плавная ease-out анимация ~280 мс, накопление повторных нажатий, прерывание ручным pan/wheel/pinch, мгновенно при `prefers-reduced-motion`).
- `services/frontend/static/app.js` — движок `StarMap`: состояние `zoomAnim`, методы `zoomSmooth`, `animateViewTo`, `applyViewAnimation`, `fitTarget` / `fitViewSmooth`; easing `easeOutCubic`; анимация вида применяется в RAF-цикле; сброс анимации в `pointerdown` / `wheel` / `mount` / `unmount`; кнопки `+`/`−`/`⤢` переведены на плавные варианты.
- `tests/unit/test_star_map.py` — новый тест `test_star_map_smooth_zoom_buttons`.
- `STATUS.md`, `CHANGE_HISTORY.md`.

### Суть изменений:
1. Кнопки `+` и `−` теперь масштабируют карту плавно (ease-out cubic, 280 мс) с якорем в центре полотна; кнопка `⤢` (вписать граф) анимирует переход к fit-виду за 340 мс.
2. Повторные быстрые нажатия накапливают зум поверх целевого значения текущей анимации и плавно перенацеливают её — без рывков и сбросов.
3. Ручное взаимодействие (drag/pan, колесо, pinch) мгновенно прерывает анимацию, чтобы не конфликтовать с прямым управлением; колесо и pinch остаются мгновенными (direct manipulation).
4. При `prefers-reduced-motion: reduce` зум кнопками применяется мгновенно.

### Верификация:
- `node --check services/frontend/static/app.js` — синтаксис валиден.
- `python3 -m unittest discover -s tests/unit -p "test_*.py"` — **36/36 тестов OK** (добавлен `test_star_map_smooth_zoom_buttons`).
- Node smoke-тест движка: интерполяция k в середине анимации, точное завершение, якорь в центре, накопление повторных кликов (1.25 → 1.5625), клампы [0.3, 3.2], плавный fit, мгновенное применение при reduced motion — OK.


## [2026-11-16] — Star Map: интерактивная карта связей SO-тредов из RAG-ответа (FR-3.4, FR-3.5, ADR-006) (Milestone 1, Iteration 88)

### Затронутые компоненты и спецификации:
- `docs/Product/03-requirements.md` — добавлены FR-3.4 (вход из карточки RAG-ответа, анимированный переход, управление картой) и FR-3.5 (режим мок-данных по контракту `graph.proto`).
- `docs/Product/04-interface-and-behavior.md` — добавлен «Экран 2.1: Star Map» (переходы, полотно, управление, hover/click, легенда, шапка).
- `docs/Tech/04-api-and-access.md` — примечание о режиме моков в §2.5 (`/api/v1/graph`).
- `docs/Tech/WORKSPACE_MAP.md` — зона ответственности фронтенда дополнена Star Map.
- `docs/Service/decisions.md` — ADR-006: Vanilla Canvas 2D + собственная force-симуляция, без D3/Cytoscape/React Flow.
- `services/frontend/static/index.html` — экран `#star-map-view` (canvas, шапка с Back to chat, легенда 4 категорий, zoom-контролы, тултип, панель деталей, подсказка); версия кэша styles.css поднята до `v=20261116_22`.
- `services/frontend/static/app.js` — кнопка `Star Map` в шапке `.assistant-card`; мок-пул `MOCK_RELATED_THREADS` (14 тредов) и билдер `buildStarMapGraph` (категории `PRIMARY_RESULT`/`OTHER_RESULT`/`DUPLICATE`/`LINKED`, ребра `DUPLICATE_OF`/`LINKED_IN_ANSWER`, поля по `ThreadNode`/`ThreadDetailsResponse`); Canvas-движок `StarMap` (отталкивание, пружины, центрирование, затухание, каскадное «рождение» узлов, звездное поле, drag/pan/wheel/pinch, hover-тултип, панель деталей); анимированные переходы `openStarMap`/`closeStarMap`/`hideStarMapImmediate`; маршрут `/star-map`, закрытие по `Esc`; i18n EN/RU.
- `services/frontend/static/styles.css` — блок Star Map в дизайн-системе Errorium (liquid glass, аура, цвета FR-3.2 `#10b981`/`#f59e0b`/`#38bdf8`/`#a855f7`), keyframes ухода чата и prompt bar и появления карты, адаптив (мобильная bottom-sheet панель деталей), `prefers-reduced-motion`.
- `tests/unit/test_star_map.py` — 11 новых unit-тестов (структура, кнопка входа, анимации, контракт моков, отсутствие внешних библиотек, цвета/размер узлов, hover/click, роутинг и Esc, дизайн-система, i18n, `node --check`).
- `STATUS.md`, `CHANGE_HISTORY.md`.

### Суть изменений:
1. **Точка входа и переход (FR-3.4):** в шапке карточки RAG-ответа появилась кнопка `Star Map` (иконка созвездия). По нажатию сообщения уходят вверх с fade + scale + blur, prompt bar уезжает вниз, и на их месте появляется полноэкранная карта с проявлением из blur/увеличения; узлы рождаются каскадом (~45 мс). Возврат (`Back to chat` / `Esc`) проигрывает обратную анимацию, переписка сохраняется.
2. **Карта в стиле Obsidian:** Canvas 2D с мерцающим звездным полем и синей аурой; собственная force-симуляция (кулоновское отталкивание, пружины ребер, центрирование, затухание, «подогрев» при drag). Цвета и размеры узлов по FR-3.2 (радиус ∝ sqrt(score)), пунктирные ребра дубликатов, подсветка связей выбранного узла с затемнением остальных.
3. **Интерактив по FR-3.3:** hover — стеклянный тултип (заголовок, сниппет, ✅, ▲ score, теги); клик — правая стеклянная панель с категорией, голосами, тегами, текстом лучшего/принятого ответа и ссылкой `Open on Stack Overflow`. Панорамирование, zoom колесом/pinch/кнопками, вписывание графа, перетаскивание узлов.
4. **Мок-данные (FR-3.5):** граф строится из источников RAG-ответа и пула связанных тредов, детерминированно по запросу; структура совместима с `ThreadGraphResponse`/`ThreadDetailsResponse` — замена на реальный `GraphService` не потребует переписывания визуализации.

### Верификация:
- `node --check services/frontend/static/app.js` — синтаксис валиден.
- `python3 -m unittest discover -s tests/unit -p "test_*.py"` — **35/35 тестов OK** (24 существующих + 11 новых Star Map).
- `go test ./...` в `services/gateway` — ok (handler, migrations, config).
- Node smoke-тест билдера графа: 14 узлов / 16 ребер, категории {PRIMARY_RESULT:1, OTHER_RESULT:1, DUPLICATE:3, LINKED:9}, детерминизм, уникальность doc_id, валидность ребер и полей контракта — OK.
- Бенчмарк не запускался: правки ранжирования и токенизации не вносились (только фронтенд).
## [2026-11-16] — Увеличение горизонтального разнесения сообщений в чате (Milestone 1, Iteration 123)

### Затронутые компоненты и спецификации:
- `services/frontend/static/css/chat.css`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Более выразительное разнесение реплик чата по краям:**
   - В [chat.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/css/chat.css) отступ для сообщений пользователя `.message-row.user` увеличен вправо до `margin-right: -24px;` с `max-width: 80%;`.
   - Ответ ассистента `.message-row.assistant` сдвинут влево до `margin-left: -28px;` и `width: calc(100% + 28px);`.
   - Диалог приобрёл яркую динамику живого мессенджера с чётким разделением сторон собеседников.
2. **Верификация и тесты:**
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).


## [2026-11-16] — Разнесение сообщений пользователя и ассистента в стороны (Milestone 1, Iteration 122)

### Затронутые компоненты и спецификации:
- `services/frontend/static/css/chat.css`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Горизонтальное смещение сообщений чата:**
   - В [chat.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/css/chat.css) для строки пользователя `.message-row.user` добавлен отступ `margin-right: -10px;` и ограничение `max-width: 82%;`, сдвигающее пузырь пользователя правее.
   - Для строки ответа ассистента `.message-row.assistant` добавлен отступ `margin-left: -14px;` и ширина `width: calc(100% + 14px);`, благодаря чему аватарка маскота и бабл ответа вынесены левее, создавая отчётливую диалоговую асимметрию между репликами.
2. **Верификация и тесты:**
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).


## [2026-11-16] — Обводка аватарки маскота в цвет границы бабла сообщения (Milestone 1, Iteration 121)

### Затронутые компоненты и спецификации:
- `services/frontend/static/css/chat.css`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Обводка аватарки маскота:**
   - В [chat.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/css/chat.css) для селектора `.assistant-avatar` установлена обводка `border: 1px solid var(--border-subtle);` в единый цвет границы карточки сообщения `var(--border-subtle)` (`rgba(255, 255, 255, 0.1)`).
2. **Верификация и тесты:**
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).


## [2026-11-16] — Существенный подъём авы маскота ассистента в чате (Milestone 1, Iteration 120)

### Затронутые компоненты и спецификации:
- `services/frontend/static/css/chat.css`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Смещение авы маскота вверх:**
   - В [chat.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/css/chat.css) для `.assistant-avatar` верхний отступ изменен на `margin-top: -18px;`, благодаря чему аватарка маскота поднята ощутимо выше относительно бабла сообщения.
2. **Верификация и тесты:**
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).


## [2026-11-16] — Удаление синей обводки контейнера аватарки ассистента и масштабирование иконки (Milestone 1, Iteration 119)

### Затронутые компоненты и спецификации:
- `services/frontend/static/css/chat.css`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Чистая иконка ассистента без внешней синей обводки:**
   - В [chat.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/css/chat.css) у контейнера `.assistant-avatar` убрана синяя граница (`border: none;`), синий полупрозрачный фон (`background: transparent;`) и внешняя тень.
   - Размер самого контейнера установлен в `34x34px`, а изображение иконки маскота занимает все 100% пространства (`width: 100%; height: 100%; object-fit: cover;`), благодаря чему сама иконка стала значительно крупнее, детальнее и выглядит чисто без лишней рамки.
2. **Верификация и тесты:**
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).


## [2026-11-16] — Упрощение названия блока источников и поднятие аватарки маскота (Milestone 1, Iteration 118)

### Затронутые компоненты и спецификации:
- `services/frontend/static/app.js`
- `services/frontend/static/css/chat.css`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Лаконичный заголовок источников без слова 'Ранжированные':**
   - В [app.js](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/app.js) в карточке решения заголовок свёрнутого блока источников изменён:
     - RU: `Источники Stack Overflow (N)` (вместо `Ранжированные источники...`).
     - EN: `Stack Overflow Sources (N)` (вместо `Ranked Stack Overflow...`).
   - Обновлены двусторонние соответствия в словаре локализации `I18N_STRINGS`.
2. **Позиция аватарки маскота:**
   - В [chat.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/css/chat.css) для селектора `.assistant-avatar` отступ сверху скорректирован на `margin-top: -6px;`. Аватарка приподнята выше и теперь аккуратно визуально выравнивается с началом текста реплики ассистента.
3. **Верификация и тесты:**
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).


## [2026-11-16] — Вынос аватарки маскота рядом с баблом ответа ИИ и скрытие плашки заголовка RAG (Milestone 1, Iteration 117)

### Затронутые компоненты и спецификации:
- `services/frontend/static/app.js`
- `services/frontend/static/css/chat.css`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Диалоговая аватарка ассистента рядом с баблом:**
   - В [app.js](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/app.js) из карточки решения убрана верхняя плашка `.card-top-bar` с надписью `Errorium RAG-синтез (BM25 Ядро)`.
   - В разметку строки ответа ассистента `.message-row.assistant` добавлен элемент `.assistant-avatar` с круглым фавиконом маскота, расположенный слева от бабла карточки.
   - В [chat.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/css/chat.css) настроена флекс-раскладка (`display: flex; flex-direction: row; gap: 12px;`): аватарка стоит рядом с левым верхним «хвостиком» карточки (`border-radius: 4px 20px 20px 20px`), создавая эффект прямой реплики говорящего персонажа.
2. **Верификация и тесты:**
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).


## [2026-11-16] — Увеличение радиуса скругления остальных углов карточки ответа ИИ (.assistant-card) (Milestone 1, Iteration 116)

### Затронутые компоненты и спецификации:
- `services/frontend/static/css/chat.css`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Более выразительное закругление углов ответа ИИ:**
   - В [chat.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/css/chat.css) для селектора `.assistant-card` значение `border-radius` обновлено до `4px 20px 20px 20px;`.
   - Левый верхний угол остался почти прямым (4px), а остальные три угла (правый верхний, правый нижний и левый нижний) стали заметно мягче и округлее (20px).
2. **Верификация и тесты:**
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).


## [2026-11-16] — Стилизация левого верхнего угла карточки ответа ИИ (.assistant-card) по аналогии с сообщением пользователя (Milestone 1, Iteration 115)

### Затронутые компоненты и спецификации:
- `services/frontend/static/css/chat.css`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Скругление углов карточки ответа ассистента:**
   - В [chat.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/css/chat.css) для селектора `.assistant-card` значение `border-radius` обновлено с `16px` на `4px 16px 16px 16px`.
   - Теперь левый верхний угол ответа ИИ имеет минимальное закругление в 4px (аналогично правому нижнему углу сообщения пользователя `.user-bubble` со скруглением `18px 18px 4px 18px`), создавая гармоничный диалоговый вид «хвостика» входящего сообщения.
2. **Верификация и тесты:**
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).


## [2026-11-16] — Замена иконки карточки RAG на фавикон и перевод ролей 'Фуллстек' и 'Инженер данных / ML' (Milestone 1, Iteration 114)

### Затронутые компоненты и спецификации:
- `services/frontend/static/app.js`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Иконка в заголовке ответа RAG-синтеза:**
   - В [app.js](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/app.js) в функции `renderConversation` в блоке `.card-top-bar .badge-mascot` изображение заменено с общего `/mascot.png` на круглый фирменный фавикон `/favicon.png`, в точности как на референсе.
2. **Уточнение и дополнение переводов ролей:**
   - В словаре локализации `I18N_STRINGS` добавлена русская форма с двумя «л» для **`Fullstack developer`** $\rightarrow$ **`Фуллстек-разработчик`** (с поддержкой обратного перевода `Фуллстек-разработчик` и `Фулстек-разработчик`).
   - Добавлен перевод роли **`Data engineer / ML`** $\rightarrow$ **`Инженер данных / ML`** (с двусторонней обратимостью).
3. **Верификация и тесты:**
   - Проверена 100% обратимость переводов (0 пропущенных ключей).
   - Все 24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).


## [2026-11-16] — Полная двусторонняя локализация всех ролей пользователя в профиле и сайдбаре (Milestone 1, Iteration 113)

### Затронутые компоненты и спецификации:
- `services/frontend/static/css/profile.css`
- `services/frontend/static/app.js`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Цвет бейджика карандаша аватарки:**
   - В [profile.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/css/profile.css) фон кружочка кнопки редактирования фото аватарки `.profile-edit-badge` установлен в чистый белый `#ffffff`, а при наведении `#f1f5f9` — в точном соответствии с цветом заголовка никнейма под аватаркой.
2. **Локализация всех профессиональных ролей:**
   - В словари `I18N_STRINGS.ru` и `I18N_STRINGS.en` в [app.js](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/app.js) добавлены полные двусторонние переводы для всех 10 ролей:
     - `Backend developer` $\leftrightarrow$ `Бэкенд-разработчик`
     - `Frontend developer` $\leftrightarrow$ `Фронтенд-разработчик`
     - `Fullstack developer` $\leftrightarrow$ `Фулстек-разработчик`
     - `DevOps engineer` $\leftrightarrow$ `DevOps-инженер`
     - `QA engineer` $\leftrightarrow$ `QA-инженер`
     - `Data engineer / ML` $\leftrightarrow$ `Data engineer / ML`
     - `System architect` $\leftrightarrow$ `Системный архитектор`
     - `Mobile developer` $\leftrightarrow$ `Мобильный разработчик`
     - `Security engineer` $\leftrightarrow$ `Инженер по безопасности`
     - `Tech lead` $\leftrightarrow$ `Тимлид / Техлид`
   - В `updateAuthUI()` и `applyLanguage()` добавлена динамическая локализация текущей роли пользователя как в сайдбаре (`#sidebar-auth-role`), так и в строке роли профиля (`#profile-info-role`).
   - TreeWalker автоматически переводит элементы выпадающего меню `.role-dropdown-item`.
3. **Верификация и тесты:**
   - Проверена 100% обратимость переводов.
   - Синтаксис проверен (`node --check`).
   - Все 24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).


## [2026-11-16] — Уменьшение размера иконок карандаша в полях профиля (Milestone 1, Iteration 112)

### Затронутые компоненты и спецификации:
- `services/frontend/static/index.html`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Компактный размер карандашей в инпутах:**
   - В [index.html](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/index.html) размер иконок `.profile-edit-icon` уменьшен с 16x16px до деликатных 13x13px (`width="13" height="13"`), благодаря чему они смотрятся ещё более легко и минималистично рядом со значениями полей никнейма и email.
2. **Верификация и тесты:**
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).


## [2026-11-16] — Добавление минималистичных иконок карандаша для полей редактирования профиля (Milestone 1, Iteration 111)

### Затронутые компоненты и спецификации:
- `services/frontend/static/index.html`
- `services/frontend/static/css/profile.css`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Иконки редактирования для никнейма и email:**
   - В [index.html](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/index.html) с правой стороны строк полей `Username` (`#profile-info-username`) и `Email` (`#profile-info-email`) добавлены SVG-иконки карандаша (`.profile-edit-icon`) размером 16x16px (в едином стиле и размере со стрелочками-шевронами в выпадающих списках).
   - В [profile.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/css/profile.css) настроена стилизация и состояния:
     - Базовый аккуратный приглушённый цвет `#8c9fb9` с `opacity: 0.65`.
     - При наведении курсора на строку подсветка до `#b8cef1` (`opacity: 0.95`).
     - При активном фокусе в поле ввода — акцентный небесно-голубой цвет `#38bdf8` (`opacity: 1`).
2. **Верификация и тесты:**
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).


## [2026-11-16] — Отключение звёздочки курсора везде, кроме лендинга (Milestone 1, Iteration 110)

### Затронутые компоненты и спецификации:
- `services/frontend/static/css/base.css`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Показ звёздочки кометы только на лендинге:**
   - В [base.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/css/base.css) базовое состояние `.cursor-comet-canvas.active` переведено в `opacity: 0;` (звёздочка и её след полностью скрыты на экранах поиска, чата, профиля и модальных окон).
   - Правило `body.landing-active .cursor-comet-canvas.active { opacity: 1; }` активирует эффект исключительно при нахождении пользователя на промо-странице лендинга.
2. **Верификация и тесты:**
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).


## [2026-11-16] — 50% прозрачность звёздочки и хвоста кометы на всех экранах кроме лендинга (Milestone 1, Iteration 109)

### Затронутые компоненты и спецификации:
- `services/frontend/static/css/base.css`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Прозрачность эффекта звёздочки кометы курсора:**
   - В [base.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/css/base.css) для класса `.cursor-comet-canvas.active` базовая прозрачность установлена в `opacity: 0.5;` (50% прозрачности на экранах поиска, чата, профиля, модальных окон и настроек).
   - При переходе на лендинг (класс `body.landing-active`) активируется правило `body.landing-active .cursor-comet-canvas.active { opacity: 1; }` с сохранением полной 100% яркости и насыщенности свечения.
2. **Верификация и тесты:**
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).


## [2026-11-16] — Уточнение толщины белой обводки аватарки в сайдбаре до 0.8px (Milestone 1, Iteration 108)

### Затронутые компоненты и спецификации:
- `services/frontend/static/css/profile.css`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Толщина обводки аватарки:**
   - В [profile.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/css/profile.css) для `.user-avatar-badge` толщина границы установлена в `0.8px solid rgba(255, 255, 255, 0.85);`.
2. **Верификация и тесты:**
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).


## [2026-11-16] — Белая обводка 1px для аватарки пользователя в сайдбаре (Milestone 1, Iteration 107)

### Затронутые компоненты и спецификации:
- `services/frontend/static/css/profile.css`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Белая рамка у аватарки:**
   - В [profile.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/css/profile.css) для селектора `.user-avatar-badge` добавлена белая обводка `border: 1px solid rgba(255, 255, 255, 0.85);` и свойство `overflow: hidden;` для идеального скругления изображения фото/буквы внутри круга.
2. **Верификация и тесты:**
   - Все 24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).


## [2026-11-16] — Отключение подсветки (свечения) у аватарки в сайдбаре (Milestone 1, Iteration 106)

### Затронутые компоненты и спецификации:
- `services/frontend/static/css/profile.css`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Удаление box-shadow у аватарки в сайдбаре:**
   - В [profile.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/css/profile.css) для селектора `.user-avatar-badge` (аватарка текущего пользователя `#sidebar-auth-avatar` в нижнем блоке сайдбара) установлено `box-shadow: none`.
   - Теперь аватарка в боковом меню отображается аккуратно без фонового ореола/свечения.
2. **Верификация и тесты:**
   - Все 24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).


## [2026-11-16] — Модульная декомпозиция CSS-стилей фронтенда на отдельные файлы (Milestone 1, Iteration 105)

### Затронутые компоненты и спецификации:
- `services/frontend/static/styles.css`
- `services/frontend/static/css/` (11 модульных файлов)
- `tests/unit/test_frontend_spa.py`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Декомпозиция монолитного файла `styles.css` (3787 строк) на функциональные модули:**
   - Создана директория `services/frontend/static/css/` с логическими модулями:
     - [base.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/css/base.css) — дизайн-токены CSS-переменных, глобальный сброс, типографика и холст звёздочки курсора.
     - [layout.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/css/layout.css) — корневой контейнер `.app-container` и фоновый оверлей `.drawer-backdrop`.
     - [sidebar.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/css/sidebar.css) — боковое меню, поиск по чатам, группировка истории, кнопка сворачивания меню.
     - [hero.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/css/hero.css) — главный экран приветствия с парящим маскотом, слоганами и синей аурой свечения.
     - [landing.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/css/landing.css) — лендинг (`/mcp` и `/about`), карточки преимуществ, плашки слогана, секции и навигация.
     - [chat.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/css/chat.css) — диалоговые пузыри пользователя (`.user-bubble`), карточки ассистента (`.assistant-card`), источники и сниппеты кода.
     - [prompt.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/css/prompt.css) — нижняя строка ввода запроса, кнопка прикрепления стек-трейса, селектор технологий.
     - [modals.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/css/modals.css) — базовые модальные окна, форма настройки MCP с копированием токена, окно About.
     - [responsive.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/css/responsive.css) — медиа-запросы адаптивности под планшеты, десктопы (`min-width: 930px`) и компактные по высоте экраны (`max-height: 720px`).
     - [profile.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/css/profile.css) — экран профиля пользователя, аватар, дропдауны ролей/языка и сетка тегов технологий.
     - [auth.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/css/auth.css) — модальные окна авторизации, регистрации, сброса пароля и алертов.
   - [styles.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/styles.css) преобразован в чистую точку входа с директивами `@import url('./css/...')`.
2. **Верификация и тесты:**
   - Обновлен тест `test_frontend_spa.py` с поддержкой загрузки модульных `@import` файлов и проверкой наличия всех 11 CSS-модулей в `test_static_files_exist`.
   - Все 24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).


## [2026-11-16] — Обновление перечня поддерживаемых AI-агентов и сред в секции MCP (Milestone 1, Iteration 104)

### Затронутые компоненты и спецификации:
- `services/frontend/static/index.html`
- `services/frontend/static/app.js`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Обновление списка сред в модальном окне и на лендинге:**
   - В [index.html](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/index.html) и словаре локализации [app.js](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/app.js) перечень сред обновлен:
     - Ранее: `Cursor, Claude Code, and Windsurf`.
     - Теперь: `Codex, Claude Code, Antigravity, and DeepSeek Harness` (в подзаголовке лендинга: `Codex, Claude Code, Antigravity, DeepSeek Harness, and custom AI agents`).
   - На русском языке: `Codex, Claude Code, Antigravity и DeepSeek Harness`.
   - Обеспечена полная обратная совместимость словаря локализации `en` <-> `ru`.
2. **Верификация и тесты:**
   - Проверена обратимость всех ключей I18N.
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).


## [2026-11-16] — Исправление обратного перевода секций лендинга на английский язык (Milestone 1, Iteration 103)

### Затронутые компоненты и спецификации:
- `services/frontend/static/app.js`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Симметричный обратный перевод RU -> EN в словаре `I18N_STRINGS.en`:**
   - В [app.js](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/app.js) в секцию `I18N_STRINGS.en` добавлены недостающие ключи для карточек лендинга и модального окна MCP:
     - `"Remote MCP через SSE": "Remote MCP over SSE"`
     - `"Точный поиск BM25": "Precise BM25 Retrieval"`
     - `"Защищенный Bearer-токен": "Secure Bearer Token"`
     - `"Персональный Remote MCP Bearer токен:": "Personal Remote MCP Bearer Token:"`
     - `"Сверхбыстрый IPC": "Sub-millisecond IPC"`
     - `"Потоковый синтез": "Streaming Synthesis"`
     - `"Насыщение частоты терминов": "Term Frequency Saturation"`
     - `"Нормализация длины документа": "Document Length Normalization"`
     - `"Потоковая выдача без задержек": "Zero-Latency Streaming Delivery"`
     - `"Агент в IDE": "IDE Agent"`
     - `"ИУ5 МГТУ": "IU5 BMSTU"`
   - Теперь при переключении с русского на английский язык ни один элемент или заголовок карточек лендинга не остаётся на русском.
2. **Верификация и тесты:**
   - Запущен скрипт полной проверки обратимости переводов: 0 отсутствующих соответствий.
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).


## [2026-11-16] — Синхронизация подсветки технических терминов в описании лендинга на английском языке (Milestone 1, Iteration 102)

### Затронутые компоненты и спецификации:
- `services/frontend/static/index.html`
- `services/frontend/static/app.js`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Подсветка ключевых терминов на английском языке:**
   - В [index.html](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/index.html) и функции динамического переключения языка `applyLanguage` в [app.js](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/app.js) в описание лендинга `.landing-intro-description` для английской версии добавлены теги `<span class="tech-highlight">` в полном соответствии с русской версией.
   - Теперь термины **`segmentation fault`**, **`runtime panic`**, **`BM25`** и **`IDE`** выделяются акцентным цветом `#93c5fd` как в русской, так и в английской версии.
2. **Верификация и тесты:**
   - Проверена корректность синтаксиса JavaScript (`node --check`).
   - Все 24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).


## [2026-11-16] — Возврат английской надписи главной кнопки 'Start debugging with Errorium' при сохранении русской 'Попробовать Errorium' (Milestone 1, Iteration 101)

### Затронутые компоненты и спецификации:
- `services/frontend/static/index.html`
- `services/frontend/static/app.js`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Текст кнопки лендинга:**
   - В [index.html](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/index.html) английский исходный текст кнопки `#btn-landing-hero-start` возвращён к исходному: `Start debugging with Errorium`.
   - В словаре локализации [app.js](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/app.js) сохранена русская версия кнопки: `Попробовать Errorium` (двустороннее сопоставление `Start debugging with Errorium` <-> `Попробовать Errorium`).
2. **Верификация и тесты:**
   - Проверена корректность синтаксиса JavaScript (`node --check`).
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).


## [2026-11-16] — Перевод заголовка и плашек подзаголовка на единый премиальный шрифт Plus Jakarta Sans (Milestone 1, Iteration 100)

### Затронутые компоненты и спецификации:
- `services/frontend/static/index.html`
- `services/frontend/static/styles.css`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Единая типографика Plus Jakarta Sans:**
   - В [index.html](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/index.html) расширен импорт весов `Plus Jakarta Sans: 400, 500, 600, 700, 800`.
   - В [styles.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/styles.css) для главного заголовка `.landing-welcome-title` и синих плашек подзаголовка `.landing-slogan-line` установлен `font-family: var(--font-editorial);` (Plus Jakarta Sans).
   - Весь первый экран теперь звучит в едином целостном визуальном ключе: от массивного жирного заголовка `Hello, this is Errorium.` до аккуратных плашек слогана и текста описания.
2. **Верификация и тесты:**
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).


## [2026-11-16] — Изменение текста главной кнопки старта на 'Попробовать Errorium' / 'Try Errorium' (Milestone 1, Iteration 99)

### Затронутые компоненты и спецификации:
- `services/frontend/static/index.html`
- `services/frontend/static/app.js`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Обновление надписи кнопки:**
   - В [index.html](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/index.html) текст кнопки `#btn-landing-hero-start` изменен со `Start debugging with Errorium` на лаконичное `Try Errorium`.
   - В [app.js](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/app.js) настроена русская локализация `Попробовать Errorium`.
2. **Верификация и тесты:**
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).


## [2026-11-16] — Скрытие звёздочки курсора при наведении на кнопки (Milestone 1, Iteration 98)

### Затронутые компоненты и спецификации:
- `services/frontend/static/app.js`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Скрытие звёздочки при наведении на кнопки:**
   - В [app.js](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/app.js) в обработчике `mousemove` функции `initCursorComet()` добавлена проверка `e.target.closest('button, [role="button"], input[type="button"], input[type="submit"]')`.
   - Когда курсор наводится на любую кнопку (`Start debugging`, копирование токена, переключатель языка, кнопки модалок и бокового меню), светящаяся звёздочка и её ореол мгновенно и чисто исчезают, не перекрывая текст и иконки на кнопке.
   - Как только курсор съезжает с кнопки обратно на фон или карточки, звёздочка плавно появляется снова.
2. **Верификация и тесты:**
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).


## [2026-11-16] — Плавное радиальное свечение вместо чёткого круга у звёздочки курсора (Milestone 1, Iteration 97)

### Затронутые компоненты и спецификации:
- `services/frontend/static/app.js`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Бесшовное радиальное свечение звёздочки:**
   - В [app.js](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/app.js) в функции `initCursorComet()` жесткий круговой контур заменен на мягкий непрерывный радиальный градиент (`ctx.createRadialGradient` с радиусом 28px).
   - Свет плавно и бесшовно затухает от яркого центра (`rgba(56, 189, 248, 0.45)`) в прозрачный фон (`rgba(56, 189, 248, 0)`), полностью устраняя эффект резкой окружности.
   - Звёздочка выглядит воздушно и сияет естественным рассеянным неоновым ореолом.
2. **Верификация и тесты:**
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).


## [2026-11-16] — Интерактивный эффект курсора в виде светящейся косической звёздочки/кометы со шлейфом (Milestone 1, Iteration 96)

### Затронутые компоненты и спецификации:
- `services/frontend/static/styles.css`
- `services/frontend/static/app.js`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Интерактивный курсор-комета / звёздочка (Cursor Comet Trail):**
   - Фонариковая статичная подсветка заменена на динамичную изящную комету со звёздочкой и световым шлейфом:
     - Во главе курсора — сверкающая четырехлучевая звёздочка с белым горячим центром и неоново-голубым ореолом (`#38bdf8`);
     - За движением курсора тянется плавный мерцающий кометный шлейф с постепенным угасанием и микро-искрами;
     - Реализовано на легковесном Canvas 2D (`initCursorComet()`), оптимизированном через физику LERP и пружинящую инерцию (`ease: 0.22`);
     - `pointer-events: none;` — не мешает кликам, ссылкам и выделению текста;
     - Автоматически засыпает при остановке курсора и не нагружает процессор.
2. **Верификация и тесты:**
   - Синтаксис проверен через `node --check services/frontend/static/app.js`.
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).


## [2026-11-16] — Деликатное повышение общей прозрачности фонарика курсора (opacity: 0.82) (Milestone 1, Iteration 95)

### Затронутые компоненты и спецификации:
- `services/frontend/static/styles.css`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Деликатная прозрачность фонарика:**
   - В [styles.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/styles.css) для активного состояния фонарика курсора `.cursor-flashlight-torch.active` прозрачность установлена на комфортные `opacity: 0.82` (вместо плотной единицы).
   - Свет стал совсем немного более полупрозрачным, утонченным и атмосферным.
2. **Верификация и тесты:**
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).


## [2026-11-16] — Рассеивающееся белое свечение и окрашивание основы фонарика в гамму логотипа Errorium (Milestone 1, Iteration 94)

### Затронутые компоненты и спецификации:
- `services/frontend/static/styles.css`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Более рассеянное белое свечение в центре:**
   - В [styles.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/styles.css) белый световой блик сделан более объемным и плавно рассеивающимся (`radial-gradient(circle at 50% 50%, rgba(255, 255, 255, 0.45) 0%, rgba(255, 255, 255, 0.2) 14%, rgba(255, 255, 255, 0.05) 28%, transparent 40%)`).
2. **Основа фонарика в точном цвете слова Errorium:**
   - Самый широкий световой ореол переведён в гамму бренда `Errorium`:
     `rgba(56, 189, 248, 0.18)` (Sky 400), `rgba(96, 165, 250, 0.12)` (Blue 400) и `rgba(147, 197, 253, 0.05)` (Blue 300).
   - Общий диаметр увеличен до `380px`, фильтр мягкого размытия скорректирован до `blur(22px)`.
3. **Верификация и тесты:**
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).


## [2026-11-16] — Увеличение яркости компактного белого ядра фонарика курсора (Milestone 1, Iteration 93)

### Затронутые компоненты и спецификации:
- `services/frontend/static/styles.css`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Увеличение яркости белого ядра:**
   - В [styles.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/styles.css) для центрального белого слоя фонарика курсора увеличена яркость:
     `radial-gradient(circle at 50% 50%, rgba(255, 255, 255, 0.5) 0%, rgba(255, 255, 255, 0.18) 7%, transparent 16%)`.
   - Точка света в центре стала заметнее и ярче, сохраняя аккуратный компактный радиус.
2. **Верификация и тесты:**
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).


## [2026-11-16] — Увеличение прозрачности фонарика курсора и уменьшение белого ядра (Milestone 1, Iteration 92)

### Затронутые компоненты и спецификации:
- `services/frontend/static/styles.css`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Деликатный баланс слоев фонарика:**
   - В [styles.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/styles.css) для `.cursor-flashlight-torch`:
     - Белое световое пятно в центре уменьшено более чем в 2 раза (радиус растворения `16%` вместо `28%`, прозрачность `0.28 / 0.08`);
     - Общая прозрачность синего огонька и голубой основы увеличена (свет стал более прозрачным, мягким и не перекрывает контрастность текста);
     - Диаметр фонарика слегка скорректирован до аккуратных `320px`.
2. **Верификация и тесты:**
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).


## [2026-11-16] — Композитный свет фонарика: голубая основа, смещенный синий огонек и белый центр (Milestone 1, Iteration 91)

### Затронутые компоненты и спецификации:
- `services/frontend/static/styles.css`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Многослойный оптический эффект фонарика:**
   - В [styles.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/styles.css) для фонарика `.cursor-flashlight-torch` сформирован сложный многослойный градиент из трех оптических слоев:
     1. **В самом центре:** маленькое яркое белое световое ядро (`radial-gradient(circle at 50% 50%, rgba(255, 255, 255, 0.45) ...)`);
     2. **Внутри основы:** более плотный синий огонек поменьше, слегка асимметрично смещенный вбок (`radial-gradient(circle at 44% 46%, rgba(37, 99, 235, 0.35) ...)`);
     3. **Широкая основа:** мягкий рассеянный лазурно-голубой ореол вокруг (`radial-gradient(circle at 50% 50%, rgba(56, 189, 248, 0.16) ...)`).
   - Оптический эффект стал глубоким, объемным и "живым", напоминая настоящий сфокусированный луч линзового фонарика.
2. **Верификация и тесты:**
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).


## [2026-11-16] — Добавление плавной инерции следования фонарика за курсором (LERP physics loop) (Milestone 1, Iteration 90)

### Затронутые компоненты и спецификации:
- `services/frontend/static/app.js`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Плавная физика и инерция курсорного фонарика:**
   - В [app.js](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/app.js) в функцию `initCursorTorch()` внедрен физический цикл линейной интерполяции (LERP) с коэффициентом плавности `ease: 0.12`.
   - Фонарик больше не привязан жестко к курсору, а мягко и кинематографично догоняет его с приятной пружинящей инерцией.
   - Анимация автоматически засыпает, когда курсор останавливается, экономя ресурсы процессора.
2. **Верификация и тесты:**
   - Синтаксис проверен через `node --check services/frontend/static/app.js`.
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).


## [2026-11-16] — Уменьшение размера светового ореола фонарика курсора (320px) (Milestone 1, Iteration 89)

### Затронутые компоненты и спецификации:
- `services/frontend/static/styles.css`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Уменьшение диаметра фонарика курсора:**
   - В [styles.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/styles.css) размеры `.cursor-flashlight-torch` уменьшены с `520px` до компактных и точных `320px` (`width: 320px; height: 320px; margin: -160px;`).
   - Размытие скорректировано до `filter: blur(20px);`, свет стал более собранным и локальным вокруг курсора.
2. **Верификация и тесты:**
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).


## [2026-11-16] — Интерактивный эффект голубого фонарика (Cursor Flashlight Torch) за курсором (Milestone 1, Iteration 88)

### Затронутые компоненты и спецификации:
- `services/frontend/static/styles.css`
- `services/frontend/static/app.js`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Интерактивный световой фонарик за курсором:**
   - В [styles.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/styles.css) добавлен класс `.cursor-flashlight-torch`:
     - Фиксированный световой ореол диаметром `520px` с центрированием в координатах курсора;
     - Радиальный градиент: `radial-gradient(circle, rgba(56, 189, 248, 0.14) 0%, rgba(37, 99, 235, 0.08) 35%, rgba(56, 189, 248, 0.03) 60%, transparent 80%)`;
     - Фильтр `filter: blur(28px);`, свойство `pointer-events: none;` (не мешает кликам и выделению);
     - Плавное появление при движении мыши и скрытие при уходе курсора с экрана (`document.mouseleave`).
   - В [app.js](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/app.js) реализована функция `initCursorTorch()`:
     - Оптимизирована через `requestAnimationFrame` и аппаратное 3D-ускорение (`translate3d`), обеспечивая 60/120 FPS без просадки производительности;
     - Не активируется на сенсорных устройствах (`(pointer: coarse)`).
2. **Верификация и тесты:**
   - Синтаксис проверен через `node --check services/frontend/static/app.js`.
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).


## [2026-11-16] — Установка цвета #93c5fd из палитры выделения слогана для терминов в описании (Milestone 1, Iteration 87)

### Затронутые компоненты и спецификации:
- `services/frontend/static/styles.css`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Гармонизация цвета технических терминов:**
   - В [styles.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/styles.css) для класса `.landing-intro-description .tech-highlight` установлен цвет **`#93c5fd`** (Soft Light Blue из градиентной палитры плашек слогана).
   - Цвет стал заметно светлее, мягче и менее резким по сравнению с `#38bdf8`, идеально сочетаясь с фоновым градиентом выделения и текстом описания.
2. **Верификация и тесты:**
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).


## [2026-11-16] — Подсветка акцентным цветом (#38bdf8) технических терминов в главном описании лендинга (Milestone 1, Iteration 86)

### Затронутые компоненты и спецификации:
- `services/frontend/static/styles.css`
- `services/frontend/static/app.js`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Снятие подсветки с карточек и перенос в главный текст описания:**
   - Из карточек убрана стилизация заголовков, возвращен стандартный вид.
   - В [styles.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/styles.css) добавлен класс `.landing-intro-description .tech-highlight` с ярким фирменным небесно-голубым оттенком `#38bdf8` и начертанием `font-weight: 500`.
   - В [app.js](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/app.js) для русскоязычной версии описания подсвечены оригинальные технические термины:
     «Любая сложная ошибка, **`segmentation fault`** или **`runtime panic`** в вашем терминале уже кем-то решена. Errorium индексирует миллионы проверенных обсуждений с кастомным ранжированием **`BM25`** высокой точности, доставляя готовые решения прямо в ваш браузер и агенту в **`IDE`**.»
   - Текст смотрится акцентно, стильно и профессионально, выделяя ключевые технические понятия в потоке речи.
2. **Верификация и тесты:**
   - Синтаксис проверен через `node --check services/frontend/static/app.js`.
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).


## [2026-11-16] — Подсветка фирменным цветом (#38bdf8) технических англоязычных терминов в заголовках карточек (Milestone 1, Iteration 85)

### Затронутые компоненты и спецификации:
- `services/frontend/static/styles.css`
- `services/frontend/static/app.js`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Цветовая подсветка непереведенных англоязычных терминов в русской версии:**
   - В [styles.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/styles.css) добавлен класс `.landing-card-title .tech-term` с ярким акцентным небесно-голубым цветом `#38bdf8` и начертанием `font-weight: 600`.
   - В [app.js](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/app.js) в функции `applyLanguage` для русской версии карточек ключевые англоязычные названия подсвечены фирменным цветом:
     - `Remote MCP` через `SSE`
     - Точный поиск `BM25`
     - Защищенный `Bearer`-токен
   - В английской версии сохраняется лаконичный белый цвет заголовков.
2. **Верификация и тесты:**
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).


## [2026-11-16] — Полная двуязычная локализация (русский язык) для всех секций и описаний лендинга (Milestone 1, Iteration 84)

### Затронутые компоненты и спецификации:
- `services/frontend/static/app.js`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Полный перевод всех описаний и блоков лендинга на русский язык:**
   - В [app.js](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/app.js) в словарь `I18N_STRINGS.ru` добавлены профессиональные инженерные переводы для всех блоков:
     - Бейдж: `Next-Gen Engineering Search` → `Поиск решений нового поколения`;
     - Заголовок: `Hello, this is` → `Привет, это`;
     - Главное описание: полный перевод абзаца об исключениях, паниках и поиске решений;
     - Секция MCP: описание интеграции с Cursor, Windsurf и Claude Code, перевод `Set up MCP` → `Настройка MCP`;
     - Описание карточек: детали транспорта через SSE, токенизации кода и персональных Bearer-токенов;
     - Секция About us: описание академической специфики ИУ5 МГТУ, парсинга 100 ГБ дампа, потокового RAG и IPC gRPC;
     - Карточки архитектуры: `Designed for Developers, Verified by Benchmarks` и пояснение формулы BM25.
   - Технические заголовки карточек из присланного скриншота (`Remote MCP через SSE`, `Точный поиск BM25`, `Защищенный Bearer-токен`, а также обозначения `100 GB`, `BM25`, `RAG`, `gRPC`) оставлены аккуратными и узнаваемыми для разработчиков.
2. **Верификация и тесты:**
   - Синтаксис проверен через `node --check services/frontend/static/app.js`.
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).


## [2026-11-16] — Установка точного порядка плашек в About us: 100 GB -> RAG -> BM25 -> gRPC (Milestone 1, Iteration 83)

### Затронутые компоненты и спецификации:
- `services/frontend/static/index.html`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Новый порядок спецификаций в About us:**
   - В [index.html](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/index.html) внутри `.landing-specs-grid` порядок карточек выставлен строго по запросу:
     1. **`100 GB`** (Stack Overflow Dump — Streamed XML parsing with custom SPIMI index construction);
     2. **`RAG`** (Streaming Synthesis — Top-K document distillation with code snippet extraction and explanations);
     3. **`BM25`** (Built From Scratch — IU5 BMSTU formula: k₁ = 1.5, b = 0.75, zero third-party search libraries);
     4. **`gRPC`** (Sub-millisecond IPC — High-performance protobuf contracts between Gateway and Core :50051).
2. **Верификация и тесты:**
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).


## [2026-11-16] — Перемещение плашки 100 GB на первую позицию слева в секции About us (100 GB -> BM25 -> gRPC -> RAG) (Milestone 1, Iteration 82)

### Затронутые компоненты и спецификации:
- `services/frontend/static/index.html`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Перемещение 100 GB в крайнее левое положение:**
   - В [index.html](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/index.html) внутри сетки спецификаций `.landing-specs-grid` порядок карточек обновлен:
     1. **`100 GB`** (Stack Overflow Dump — Streamed XML parsing with custom SPIMI index construction);
     2. **`BM25`** (Built From Scratch — IU5 BMSTU formula: k₁ = 1.5, b = 0.75, zero third-party search libraries);
     3. **`gRPC`** (Sub-millisecond IPC — High-performance protobuf contracts between Gateway and Core :50051);
     4. **`RAG`** (Streaming Synthesis — Top-K document distillation with code snippet extraction and explanations).
2. **Верификация и тесты:**
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).


## [2026-11-16] — Увеличение высоты градиентной растяжки затемнения снизу вверх (height: 220px) (Milestone 1, Iteration 81)

### Затронутые компоненты и спецификации:
- `services/frontend/static/styles.css`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Увеличение высоты вертикальной растяжки:**
   - В [styles.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/styles.css) высота псевдоэлемента `.landing-standalone-page::after` увеличена со `120px` до `220px`.
   - Градиент настроен с плавным многошаговым переходом:
     `linear-gradient(to top, rgba(9, 13, 23, 0.75) 0%, rgba(9, 13, 23, 0.4) 40%, rgba(9, 13, 23, 0.15) 70%, transparent 100%)`.
   - Растяжка стала значительно выше по вертикали, сохраняя мягкую прозрачность и бесшовное растворение контента внизу экрана.
2. **Верификация и тесты:**
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).


## [2026-11-16] — Увеличение прозрачности нижней градиентной растяжки (Milestone 1, Iteration 80)

### Затронутые компоненты и спецификации:
- `services/frontend/static/styles.css`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Более прозрачная растяжка затемнения:**
   - В [styles.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/styles.css) для псевдоэлемента `.landing-standalone-page::after` градиент сделан более прозрачным и деликатным:
     `linear-gradient(to top, rgba(9, 13, 23, 0.75) 0%, rgba(9, 13, 23, 0.35) 45%, transparent 100%)`, высота скорректирована до `120px`.
   - Затухание внизу экрана стало легким и полупрозрачным, не перекрывая текст под ним.
2. **Верификация и тесты:**
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).


## [2026-11-16] — Изменение порядка плашек в секции About us: RAG -> BM25 -> 100 GB -> gRPC (Milestone 1, Iteration 79)

### Затронутые компоненты и спецификации:
- `services/frontend/static/index.html`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Новый порядок спецификаций в сетке About us:**
   - В [index.html](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/index.html) внутри сетки `.landing-specs-grid` порядок следования карточек изменен строго слева направо:
     1. **`RAG`** (Streaming Synthesis — Top-K document distillation with code snippet extraction and explanations);
     2. **`BM25`** (Built From Scratch — IU5 BMSTU formula: k₁ = 1.5, b = 0.75, zero third-party search libraries);
     3. **`100 GB`** (Stack Overflow Dump — Streamed XML parsing with custom SPIMI index construction);
     4. **`gRPC`** (Sub-millisecond IPC — High-performance protobuf contracts between Gateway and Core :50051).
2. **Верификация и тесты:**
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).


## [2026-11-16] — Добавление градиентной растяжки затемнения снизу вверх на лендинге (.landing-standalone-page::after) (Milestone 1, Iteration 78)

### Затронутые компоненты и спецификации:
- `services/frontend/static/styles.css`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Градиентная растяжка затемнения снизу экрана:**
   - В [styles.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/styles.css) для контейнера `.landing-standalone-page` добавлен псевдоэлемент `::after`:
     - Расположен внизу экрана (`bottom: 0; left: 0; right: 0; height: 140px;`);
     - Плавный градиент снизу вверх: `linear-gradient(to top, rgba(9, 13, 23, 0.95) 0%, rgba(9, 13, 23, 0.6) 45%, transparent 100%)`;
     - `pointer-events: none;` — не блокирует клики и взаимодействие с контентом;
     - Создаёт красивый эффект кинематографичного затухания страницы к нижнему краю экрана.
2. **Верификация и тесты:**
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).


## [2026-11-16] — Поднятие контента лендинга еще на 10px выше (padding-top: 84px) (Milestone 1, Iteration 77)

### Затронутые компоненты и спецификации:
- `services/frontend/static/styles.css`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Поднятие контента лендинга еще на 10px выше:**
   - В [styles.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/styles.css) верхний внутренний отступ контейнера `.landing-scroll-body` уменьшен с `94px` до `84px`.
2. **Верификация и тесты:**
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).


## [2026-11-16] — Корректировка верхнего отступа контента лендинга на 10px выше (padding-top: 94px) (Milestone 1, Iteration 76)

### Затронутые компоненты и спецификации:
- `services/frontend/static/styles.css`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Поднятие контента лендинга на 10px выше:**
   - В [styles.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/styles.css) верхний внутренний отступ контейнера `.landing-scroll-body` скорректирован со `104px` до `94px`.
2. **Верификация и тесты:**
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).


## [2026-11-16] — Увеличение верхнего отступа контента лендинга (padding-top: 104px) (Milestone 1, Iteration 75)

### Затронутые компоненты и спецификации:
- `services/frontend/static/styles.css`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Смещение контента лендинга ниже:**
   - В [styles.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/styles.css) верхний внутренний отступ контейнера `.landing-scroll-body` увеличен с `72px` до `104px`.
   - Вся страница лендинга (приветственный бейдж, заголовок Hello, плашки подзаголовка, описание и кнопки) теперь опустилась ниже относительно хедера, создавая больше свободного пространства и ощущение легкости.
2. **Верификация и тесты:**
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).


## [2026-11-16] — Возврат точного оттенка градиента надписи Errorium для плашек подзаголовка (Milestone 1, Iteration 74)

### Затронутые компоненты и спецификации:
- `services/frontend/static/styles.css`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Откат к предыдущему цвету плашек:**
   - В [styles.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/styles.css) для класса `.landing-slogan-line` возвращен точно исходный гармоничный градиент слова `Errorium`:
     `linear-gradient(135deg, rgba(96, 165, 250, 0.45) 0%, rgba(56, 189, 248, 0.5) 50%, rgba(147, 197, 253, 0.45) 100%)`.
2. **Верификация и тесты:**
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).


## [2026-11-16] — Смещение оттенка плашек подзаголовка чуть глубже в синий тон (Milestone 1, Iteration 73)

### Затронутые компоненты и спецификации:
- `services/frontend/static/styles.css`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Тонкая корректировка тональности плашек:**
   - В [styles.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/styles.css) градиент `.landing-slogan-line` сдвинут чуть глубже в синий спектр:
     `linear-gradient(135deg, rgba(59, 130, 246, 0.52) 0%, rgba(37, 140, 244, 0.52) 50%, rgba(96, 165, 250, 0.48) 100%)`.
   - Тон стал чуть насыщеннее и благороднее, сохранив мягкую полупрозрачность и связь с палитрой `Errorium`.
2. **Верификация и тесты:**
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).


## [2026-11-16] — Установка цвета и градиента как у логотипа Errorium для плашек подзаголовка (Milestone 1, Iteration 72)

### Затронутые компоненты и спецификации:
- `services/frontend/static/styles.css`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Цвет и градиент в тон слова Errorium:**
   - В [styles.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/styles.css) фон плашек `.landing-slogan-line` переведён на полупрозрачный градиент в точности по палитре слова `Errorium`:
     `linear-gradient(135deg, rgba(96, 165, 250, 0.45) 0%, rgba(56, 189, 248, 0.5) 50%, rgba(147, 197, 253, 0.45) 100%)`.
   - Плашки теперь идеально рифмуются по цвету с главным брендовым логотипом в заголовке.
2. **Верификация и тесты:**
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).


## [2026-11-16] — Полное удаление свечения (box-shadow: none) у плашек подзаголовка (Milestone 1, Iteration 71)

### Затронутые компоненты и спецификации:
- `services/frontend/static/styles.css`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Полный отказ от свечения плашек:**
   - В [styles.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/styles.css) для класса `.landing-slogan-line` удалена подсветка (`box-shadow: none;`).
   - Полупрозрачные синие прямоугольники отображаются абсолютно матово и чисто.
2. **Верификация и тесты:**
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).


## [2026-11-16] — Ослабление белого свечения плашек подзаголовка до еле заметного флёра (box-shadow: 0 0 10px rgba(255, 255, 255, 0.08)) (Milestone 1, Iteration 70)

### Затронутые компоненты и спецификации:
- `services/frontend/static/styles.css`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Снижение интенсивности свечения плашек:**
   - В [styles.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/styles.css) для класса `.landing-slogan-line` белое свечение ослаблено до тонкого полупрозрачного флёра:
     `box-shadow: 0 0 10px rgba(255, 255, 255, 0.08);`.
   - Подсветка едва уловима глазом, лишь слегка смягчает границы плашек относительно темного фона.
2. **Верификация и тесты:**
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).


## [2026-11-16] — Добавление деликатного белого свечения у полупрозрачных плашек подзаголовка (Milestone 1, Iteration 69)

### Затронутые компоненты и спецификации:
- `services/frontend/static/styles.css`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Деликатное белое свечение плашек:**
   - В [styles.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/styles.css) для класса `.landing-slogan-line` добавлено мягкое рассеянное белое свечение:
     `box-shadow: 0 0 16px rgba(255, 255, 255, 0.18), 0 0 32px rgba(255, 255, 255, 0.08);`.
   - Плашки выделения получили утонченный стеклянно-световой контур без излишней резкости.
2. **Верификация и тесты:**
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).


## [2026-11-16] — Установка полупрозрачного синего фона (rgba(37, 99, 235, 0.55)) для плашек подзаголовка (Milestone 1, Iteration 68)

### Затронутые компоненты и спецификации:
- `services/frontend/static/styles.css`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Полупрозрачный фон прямоугольников:**
   - В [styles.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/styles.css) для класса `.landing-slogan-line` цвет фона изменен с непрозрачного `#2563eb` на полупрозрачный `rgba(37, 99, 235, 0.55)`.
   - Плашки теперь пропускают темную подложку страницы, создавая легкий и естественный эффект системного выделения текста.
2. **Верификация и тесты:**
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).


## [2026-11-16] — Уменьшение вертикальных внутренних отступов (padding: 2px 10px) у плашек подзаголовка (Milestone 1, Iteration 67)

### Затронутые компоненты и спецификации:
- `services/frontend/static/styles.css`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Уменьшение вертикальных отступов плашек выделения:**
   - В [styles.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/styles.css) для класса `.landing-slogan-line` отступы уменьшены с `6px 12px` до компактных `2px 10px;`.
   - Скорректирован `line-height: 1.25;` и скругление углов `border-radius: 4px;` для идеальной высоты плашек под размер текста.
2. **Верификация и тесты:**
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).


## [2026-11-16] — Удаление свечения (box-shadow: none) у прямоугольников подзаголовка (Milestone 1, Iteration 66)

### Затронутые компоненты и спецификации:
- `services/frontend/static/styles.css`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Удаление эффекта свечения:**
   - В [styles.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/styles.css) для класса `.landing-slogan-line` удалена тень (`box-shadow: none;`).
   - Прямоугольники теперь выглядят матово, строго и чисто, как на скриншоте системного выделения текста.
2. **Верификация и тесты:**
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).


## [2026-11-16] — Разделение подзаголовка на отдельные синие прямоугольники для каждой строки (Milestone 1, Iteration 65)

### Затронутые компоненты и спецификации:
- `services/frontend/static/index.html`
- `services/frontend/static/styles.css`
- `services/frontend/static/app.js`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Отдельные прямоугольники для каждой строки:**
   - В [index.html](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/index.html) и [app.js](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/app.js) подзаголовок разбит на две дискретные строки через элементы `<span class="landing-slogan-line">`.
   - В [styles.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/styles.css) для `.landing-slogan-line` задано:
     - `display: inline-block;` и `width: fit-content;`;
     - `background-color: #2563eb;` с индивидуальным скруглением `border-radius: 6px;`, отступами `padding: 6px 12px;` и тенью `box-shadow: 0 4px 18px rgba(37, 99, 235, 0.4);`;
     - Контейнер `.landing-hero-subtitle-box` с `align-items: flex-start; gap: 8px;` обеспечивает ровный визуальный стек из двух независимых синих прямоугольников под заголовком.
2. **Верификация и тесты:**
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).


## [2026-11-16] — Стилизация подзаголовка лендинга в виде дизайнерского эффекта выделения текста маркером (Milestone 1, Iteration 64)

### Затронутые компоненты и спецификации:
- `services/frontend/static/styles.css`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Дизайнерский эффект выделения текста (Highlight / Selection Effect):**
   - В [styles.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/styles.css) для класса подзаголовка `.landing-slogan-secondary` реализован эффект выделения синим маркером в точности по присланному скриншоту:
     - `background-color: #2563eb;` (насыщенный синий цвет системного выделения);
     - `display: inline;` и `box-decoration-break: clone;` (включая вендорный префикс `-webkit-box-decoration-break: clone;`), благодаря чему каждая строка при многострочном переносе получает собственные аккуратные отступы и закругления краев;
     - `padding: 3px 8px;`, `border-radius: 4px;` и деликатное неоновое свечение `box-shadow: 0 2px 14px rgba(37, 99, 235, 0.45);`;
     - `color: #ffffff;` с белым контрастным начертанием.
2. **Верификация и тесты:**
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).


## [2026-11-16] — Подключение премиального шрифта Plus Jakarta Sans для главного описания лендинга (Milestone 1, Iteration 63)

### Затронутые компоненты и спецификации:
- `services/frontend/static/index.html`
- `services/frontend/static/styles.css`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Подключение и настройка гарнитуры Plus Jakarta Sans:**
   - В [index.html](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/index.html) в Google Fonts подключена элегантная гарнитура `Plus Jakarta Sans` (веса 400, 500, 600).
   - В [styles.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/styles.css) зарегистрирована переменная `--font-editorial: "Plus Jakarta Sans", "Inter", sans-serif;`.
   - Для блока `.landing-intro-description` применен стиль `font-family: var(--font-editorial);`, `font-size: 1.15rem;`, `line-height: 1.75;`.
   - Текст описания приобрел мягкий, чистый, современный продуктовый вид премиального AI-сервиса (в эстетике Linear / современные дизайн-системы), легко воспринимается и гармонично сочетается с заголовком и подзаголовком.
2. **Верификация и тесты:**
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).


## [2026-11-16] — Перевод шрифта большого описания лендинга с моноширинного на гарнитуру Inter (--font-ui) (Milestone 1, Iteration 62)

### Затронутые компоненты и спецификации:
- `services/frontend/static/styles.css`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Перевод описания на гарнитуру Inter:**
   - В [styles.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/styles.css) для класса `.landing-intro-description` шрифт заменен с моноширинного `var(--font-mono)` на чистый современный гротеск `var(--font-ui)` (Inter / sans-serif).
   - Отрегулирован кегль (`font-size: 1.12rem`) и межстрочный интервал (`line-height: 1.7`) для комфортного и легкого чтения объемного абзаца текста.
2. **Верификация и тесты:**
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).


## [2026-11-16] — Добавление автоподсветки активного якоря в хедере лендинга при скролле (Scrollspy) (Milestone 1, Iteration 61)

### Затронутые компоненты и спецификации:
- `services/frontend/static/index.html`
- `services/frontend/static/app.js`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Интеллектуальный Scrollspy для навигации хедера лендинга:**
   - В [app.js](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/app.js) в обработчик скролла лендинга `.landing-scroll-body` добавлено отслеживание текущей видимой секции (`#mcp` и `#about`).
   - При долистывании до соответствующего блока кнопка в хедере (`MCP` или `About us`) автоматически получает класс `.active` с фирменной голубой подсветкой (`#38bdf8`, легкий фон `rgba(56, 189, 248, 0.12)` и начертание `600`).
   - Если пользователь находится на самом верху (в приветственной части с маскотом и заголовком `Hello, this is Errorium`), активный фокус мягко снимается, сигнализируя, что открыт обзор первого экрана.
2. **Верификация и тесты:**
   - Синтаксис проверен через `node --check services/frontend/static/app.js`.
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).


## [2026-11-16] — Добавление светлой обводки в 1px для выпадающего меню выбора языка (Milestone 1, Iteration 60)

### Затронутые компоненты и спецификации:
- `services/frontend/static/styles.css`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Светлая тонкая обводка выпадающего меню:**
   - В [styles.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/styles.css) для выпадающего меню языка `.landing-lang-dropdown` добавлена деликатная светлая обводка толщиной 1px: `border: 1px solid rgba(255, 255, 255, 0.2);`.
   - Меню четко очерчено на темном фоне, сохраняя элегантный glassmorphism-стиль с мягким неоновым свечением.
2. **Верификация и тесты:**
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).


## [2026-11-16] — Удаление фоновой заливки при наведении и выборе языка в выпадающем списке (Milestone 1, Iteration 59)

### Затронутые компоненты и спецификации:
- `services/frontend/static/styles.css`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Удаление заливки прямоугольника опций языка:**
   - В [styles.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/styles.css) для состояний `.landing-lang-option:hover` и `.landing-lang-option.active` фоновая заливка убрана (`background: transparent;`).
   - Активный или наведенный пункт теперь выделяется исключительно чистым голубым цветом текста `#38bdf8` и начертанием `font-weight: 600`, без цветных плашек и прямоугольников.
2. **Верификация и тесты:**
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).


## [2026-11-16] — Увеличение скругления кнопки Start debugging до pill-формы (border-radius: 9999px) (Milestone 1, Iteration 58)

### Затронутые компоненты и спецификации:
- `services/frontend/static/styles.css`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Увеличение скругления кнопки старта:**
   - В [styles.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/styles.css) для кнопки `.landing-hero-cta-btn` значение `border-radius` изменено с `14px` на плавную современную капсульную форму `9999px` (Pill shape), а горизонтальный padding увеличен до `36px`.
   - Кнопка гармонично перекликается со скругленным бейджем `Next-Gen Engineering Search` и логотипом.
2. **Верификация и тесты:**
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).


## [2026-11-16] — Удаление хвоста 'for Cursor & terminal' из подзаголовка лендинга (Milestone 1, Iteration 57)

### Затронутые компоненты и спецификации:
- `services/frontend/static/index.html`
- `services/frontend/static/app.js`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Удаление хвоста фразы:**
   - В [index.html](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/index.html) и [app.js](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/app.js) из подзаголовка `.landing-slogan-secondary` удален фрагмент `for Cursor & terminal` (и русское `для Cursor и терминала`).
   - Фраза теперь читается емко и чисто:
     > `Stack overflow. Issue overknown — instant engineering solutions & RAG synthesis across 100 GB knowledge base.`
     > (на русском: `Стек переполнен, разраб преисполнен — поиск готовых инженерных решений и RAG-синтез по 100 ГБ базе знаний.`)
2. **Верификация и тесты:**
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).


## [2026-11-16] — Установка чистого белого цвета (#ffffff) для всей надписи подзаголовка лендинга (Milestone 1, Iteration 56)

### Затронутые компоненты и спецификации:
- `services/frontend/static/styles.css`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Чистый белый цвет подзаголовка:**
   - В [styles.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/styles.css) для класса `.landing-slogan-secondary` и внутреннего `.tagline-highlight` установлен прямо чистый белый цвет **`#ffffff`**.
   - Весь подзаголовок теперь монолитно белый, яркий, контрастный и отчетливо читаемый.
2. **Верификация и тесты:**
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).


## [2026-11-16] — Установка шрифта JetBrains Mono для главной кнопки Start debugging with Errorium (Milestone 1, Iteration 55)

### Затронутые компоненты и спецификации:
- `services/frontend/static/styles.css`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Шрифт JetBrains Mono для кнопки действия:**
   - В [styles.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/styles.css) для класса кнопки `.landing-hero-cta-btn` установлен моноширинный шрифт `font-family: var(--font-mono);` (JetBrains Mono), как и у описания лендинга.
   - Скорректированы параметры `font-size: 1.05rem;`, `font-weight: 600;` и `letter-spacing: -0.015em;` для идеального визуального баланса и четкости текста в кнопке.
2. **Верификация и тесты:**
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).


## [2026-11-16] — Смягчение яркости оттенка подзаголовка лендинга до сдержанного джинсово-синего (#5ea8d8) (Milestone 1, Iteration 54)

### Затронутые компоненты и спецификации:
- `services/frontend/static/styles.css`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Уменьшение яркости оттенка:**
   - В [styles.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/styles.css) цвет `.landing-slogan-secondary` и `.tagline-highlight` изменен с насыщенного лазурного `#7dd3fc` на спокойный, приглушенный серо-голубой/джинсовый тон **`#5ea8d8`**.
   - Текст перестал быть излишне светящимся, выглядит солидно, сдержанно и при этом отчетливо читается, сохраняя цветовую индивидуальность (не серый и не белый).
2. **Верификация и тесты:**
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).


## [2026-11-16] — Установка единого мягкого лазурного оттенка (#7dd3fc) для всей фразы подзаголовка лендинга (Milestone 1, Iteration 53)

### Затронутые компоненты и спецификации:
- `services/frontend/static/styles.css`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Единый живой цвет для всего подзаголовка:**
   - В [styles.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/styles.css) для класса `.landing-slogan-secondary` и внутреннего `.tagline-highlight` установлен единый гармоничный оттенок **`#7dd3fc`** (Sky 300 / Soft Cyan-Blue).
   - Подзаголовок выглядит ярко и читаемо на темном фоне, не теряясь как серый и не будучи ослепляюще белым, при этом идеально дополняет градиентный логотип `Errorium` и маскота.
2. **Верификация и тесты:**
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).


## [2026-11-16] — Увеличение размера подзаголовка и прижатие его ближе к Hello заголовку (Milestone 1, Iteration 52)

### Затронутые компоненты и спецификации:
- `services/frontend/static/styles.css`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Увеличение размера подзаголовка:**
   - В [styles.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/styles.css) для подзаголовка `.landing-slogan-secondary` размер шрифта увеличен с `1.15rem` до адаптивного `clamp(1.25rem, 2.2vw, 1.45rem)`.
   - Начертание усилено до выразительного полужирного `font-weight: 600`, а акцентный текст `.tagline-highlight` окрашен в контрастный `#f1f5f9`.
2. **Прижатие подзаголовка ближе к заголовку Hello:**
   - Для контейнера `.landing-hero-subtitle-box` верхний отрицательный отступ скорректирован с `-4px` до `-14px`, за счет чего подзаголовок теперь плотно и естественно примыкает к главному заголовку `Hello, this is Errorium.`.
3. **Верификация и тесты:**
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).


## [2026-11-16] — Удаление надписи Element 404 на первом экране и унификация стилей переключателя языка с кнопками навигации хедера (Milestone 1, Iteration 51)

### Затронутые компоненты и спецификации:
- `services/frontend/static/index.html`
- `services/frontend/static/styles.css`
- `services/frontend/static/app.js`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Удаление надписи Element 404:**
   - В [index.html](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/index.html) удален блок `.landing-slogan-primary` с фразой `Element 404. Reacts with developers.`.
   - В [app.js](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/app.js) удалена ветка обновления этого элемента, сразу выводится и переключается содержательный слоган:
     `Stack overflow. Issue overknown — instant engineering solutions & RAG synthesis across 100 GB knowledge base for Cursor & terminal.` (на английском) и `Стек переполнен, разраб преисполнен — поиск готовых инженерных решений и RAG-синтез по 100 ГБ базе знаний для Cursor и терминала.` (на русском).
2. **Унификация стилей переключателя языка (`.landing-lang-btn`) с кнопками MCP и About us:**
   - В [styles.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/styles.css) параметры `.landing-lang-btn` приведены в точное соответствие с `.landing-anchor-link`:
     - `padding: 7px 16px;` (ранее было `6px 12px`);
     - `font-size: 0.95rem;` (ранее было `0.88rem`);
     - `font-weight: 500;` с переходом на `font-weight: 600;` и цвет `#38bdf8` при hover/active;
     - `background: rgba(56, 189, 248, 0.12);` при hover (тот же небесно-голубой эффект подсветки);
     - Пункты выпадающего списка `.landing-lang-option` также приведены к `font-size: 0.95rem;` и аналогичному padding/hover.
3. **Верификация и тесты:**
   - Синтаксис JS проверен: `node --check services/frontend/static/app.js`.
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).


## [2026-11-16] — Темно-бордовый цвет слогана Element 404 и обновление названия раздела на Set up MCP (Milestone 1, Iteration 50)

### Затронутые компоненты и спецификации:
- `services/frontend/static/styles.css`
- `services/frontend/static/index.html`
- `services/frontend/static/app.js`
- `tests/unit/test_frontend_spa.py`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Темно-бордовый цвет слогана:**
   - В [styles.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/styles.css) для строки `.landing-slogan-primary` и её внутреннего элемента `.error-code-red` установлен единый темно-бордовый оттенок `#991b1b` (темный бордовый/ruby-red). Вся фраза `Element 404. Reacts with developers.` (включая число `404`) теперь монолитно окрашена в этот цвет.
2. **Переименование Model Context Protocol (MCP) в Set up MCP:**
   - В [index.html](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/index.html) заголовки секции интеграции MCP на лендинге (`.landing-title`) и в модальном окне (`#mcp-modal-title`) обновлены с `Model Context Protocol (MCP)` на `Set up MCP`.
   - В [app.js](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/app.js) добавлены двуязычные переводы (`Set up MCP` <-> `Настройка MCP`).
3. **Верификация и тесты:**
   - Обновлен модульный тест в [test_frontend_spa.py](file:///home/mariam/WebstormProjects/Errorium/tests/unit/test_frontend_spa.py).
   - Все 24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).


## [2026-11-16] — Удаление обводки у кнопки и выпадающего списка языка в хедере и обеспечение глобальной смены языка во всем приложении (Milestone 1, Iteration 49)

### Затронутые компоненты и спецификации:
- `services/frontend/static/styles.css`
- `services/frontend/static/app.js`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Удаление обводки у переключателя языка:**
   - В [styles.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/styles.css) у кнопки выбора языка `.landing-lang-btn` и у самого выпадающего списка `.landing-lang-dropdown` удалена граница (`border: none;`). Элемент теперь выглядит чисто, без лишних рамок, сохраняя стильную полупрозрачную подсветку при наведении и аккуратную тень списка.
2. **Исправление работы и глобальная синхронизация языка:**
   - В [app.js](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/app.js) исправлена ошибка, из-за которой не срабатывал клик по пункту меню языка (вызывалась несуществующая функция `saveAuthUser` вместо `setAuthUser`).
   - Добавлен независимый ключ сохранения языка в `localStorage` (`errorium_app_language`), благодаря чему выбранный язык сохраняется как для авторизованных пользователей, так и для гостей.
   - Метод `applyLanguage(lang)` синхронизирует состояние активного языка во всех элементах интерфейса (в лендинге, в выпадающем списке профиля, в слоганах героя и лендинга, в подсказках, в роли пользователя).
   - Выбор языка в хедере мгновенно переключает весь интерфейс приложения (чат, профиль, настройки, модальные окна, лендинг) и сохраняется при навигации по любым разделам SPA.
3. **Верификация и тесты:**
   - Синтаксис проверен через `node --check services/frontend/static/app.js`.
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).


## [2026-11-16] — Унификация размера и веса шрифта для обеих строк слогана (1.15rem, font-weight 500) (Milestone 1, Iteration 48)

### Затронутые компоненты и спецификации:
- `services/frontend/static/styles.css`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Единый размер и вес шрифта слогана:**
   - В [styles.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/styles.css) для обоих блоков `.landing-slogan-primary` и `.landing-slogan-secondary` задан строго одинаковый размер `font-size: 1.15rem` и одинаковый вес шрифта `font-weight: 500 (Medium)`.
   - Внутренние акцентные элементы (`.error-code-red` и `.tagline-highlight`) приведены к единому весу `500`.
   - Текст воспринимается аккуратно, монолитно и выдержан в едином ритме.
2. **Верификация и тесты:**
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).

## [2026-11-16] — Удаление кнопки Explore MCP & Architecture и увеличение логотипа Errorium в хедере до 1.35rem (Milestone 1, Iteration 47)

### Затронутые компоненты и спецификации:
- `services/frontend/static/index.html`
- `services/frontend/static/styles.css`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Удаление кнопки Explore MCP & Architecture:**
   - В [index.html](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/index.html) удалена вспомогательная кнопка со стрелкой `.landing-hero-learn-link`. На первом экране осталась единственная главная крупная кнопка действия `Start debugging with Errorium`.
2. **Увеличение логотипа Errorium в хедере:**
   - В [styles.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/styles.css) размер шрифта для `.landing-header-logo` увеличен с `1rem` до сбалансированных `1.35rem` (21.6px), а начертание усилено до `font-weight: 700`.
3. **Верификация и тесты:**
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).

## [2026-11-16] — Удаление плитки таблицы элементов и оформление слогана как подписи сразу после Hello (Milestone 1, Iteration 46)

### Затронутые компоненты и спецификации:
- `services/frontend/static/index.html`
- `services/frontend/static/styles.css`
- `services/frontend/static/app.js`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Удаление таблицы элементов и интеграция слогана в заголовок:**
   - В [index.html](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/index.html) полностью удален блок `.landing-element-showcase` с плиткой.
   - Слоган и расширенное описание сути перенесены сразу под заголовок `Hello, this is Errorium.` в виде элегантной подписи `.landing-hero-subtitle-box`:
     - Строка 1: `Element 404. Reacts with developers.` (`.landing-slogan-primary` с акцентным красным `404`);
     - Строка 2: `Stack overflow. Issue overknown — instant engineering solutions & RAG synthesis across 100 GB knowledge base for Cursor & terminal.` (`.landing-slogan-secondary`).
   - Ниже следует параграф `.landing-intro-description` (шрифт JetBrains Mono) и крупная кнопка старта.
   - В [app.js](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/app.js) добавлена двуязычная локализация для обеих строк слогана.
2. **Верификация и тесты:**
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).

## [2026-11-16] — Добавление выпадающего переключателя языка в правый угол хедера лендинга (Milestone 1, Iteration 45)

### Затронутые компоненты и спецификации:
- `services/frontend/static/index.html`
- `services/frontend/static/styles.css`
- `services/frontend/static/app.js`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Переключатель языка в хедере (.landing-lang-selector):**
   - В [index.html](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/index.html) в правый угол хедера добавлен интерактивный селектор языка (`English / Русский`) со стрелочкой и выпадающим списком.
   - Пока страница находится наверху и логотип `Errorium` скрыт, селектор отображается в правом углу.
   - Как только пользователь прокручивает страницу вниз и заголовок скрывается под хедером, селектор плавно растворяется (`.landing-lang-selector.hidden`), а на его месте кинематографично проявляется логотип `Errorium`. При обратном скролле наверх селектор языка плавно возвращается.
   - Выбор языка в выпадашке моментально переводит весь интерфейс и текст слогана.
2. **Верификация и тесты:**
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).

## [2026-11-16] — Двуязычная адаптация (RU/EN) слогана и описания сути платформы (Milestone 1, Iteration 44)

### Затронутые компоненты и спецификации:
- `services/frontend/static/index.html`
- `services/frontend/static/app.js`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Две полноценные языковые версии слогана-описания (#element-slogan-tagline):**
   - **Английская версия (EN):**
     `Stack overflow. Issue overknown — instant engineering solutions & RAG synthesis across 100 GB knowledge base for Cursor & terminal.`
   - **Русская версия (RU):**
     `Стек переполнен, разраб преисполнен — поиск готовых инженерных решений и RAG-синтез по 100 ГБ базе знаний для Cursor и терминала.`
   - В [app.js](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/app.js) в функцию `applyLanguage(lang)` добавлено динамическое переключение содержимого элемента `#element-slogan-tagline` в зависимости от языка пользователя.
2. **Верификация и тесты:**
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).

## [2026-11-16] — Уменьшение размера логотипа Errorium в хедере лендинга до 1rem (Milestone 1, Iteration 43)

### Затронутые компоненты и спецификации:
- `services/frontend/static/styles.css`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Размер логотипа в хедере (.landing-header-logo):**
   - В [styles.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/styles.css) размер шрифта для `Errorium` в хедере уменьшен с `1.75rem` до аккуратных `1rem` (16px).
   - Надпись стала минималистичной, пропорциональной и визуально сбалансированной относительно соседних элементов навигации.
2. **Верификация и тесты:**
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).

## [2026-11-16] — Синхронизация цвета логотипа Errorium в хедере с кнопками навигации (#94a3b8) (Milestone 1, Iteration 42)

### Затронутые компоненты и спецификации:
- `services/frontend/static/styles.css`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Цвет надписи Errorium в хедере (.landing-header-logo):**
   - В [styles.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/styles.css) цвет надписи `Errorium` изменен на `#94a3b8` — в точный цвет неактивных кнопок `MCP` и `About us`.
   - При наведении курсора надпись плавно переходит в акцентный лазурный `#38bdf8` с деликатным свечением.
2. **Верификация и тесты:**
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).

## [2026-11-16] — Расширение слогана «Stack overflow. Issue overknown» ёмким описанием сути проекта (Milestone 1, Iteration 41)

### Затронутые компоненты и спецификации:
- `services/frontend/static/index.html`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Текст слогана в шоукейсе Element 404:**
   - В [index.html](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/index.html) строка слогана дополнена емким и точным определением продукта: `Stack overflow. Issue overknown — поиск готовых инженерных решений и RAG-синтез по 100 ГБ базе знаний для Cursor и терминала.`.
   - Фраза раскрывает смысл игры слов слогана и сразу объясняет назначение платформы.
2. **Верификация и тесты:**
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).

## [2026-11-16] — Осветление голубого тона логотипа в хедере и добавление деликатного центрального неонового свечения (Milestone 1, Iteration 40)

### Затронутые компоненты и спецификации:
- `services/frontend/static/styles.css`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Светлый небесно-голубой тон логотипа в тон маскота:**
   - В [styles.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/styles.css) градиент логотипа `.landing-header-logo` осветлен до тонов маскота: `linear-gradient(135deg, #bae6fd 0%, #7dd3fc 45%, #38bdf8 100%)`.
   - Добавлен легкий центральный эффект неонового свечения `drop-shadow(0 0 10px rgba(56, 189, 248, 0.35))` со смещением строго `0 0`, создающий мягкий ореол вокруг букв без искажения контуров.
2. **Верификация и тесты:**
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).

## [2026-11-16] — Голубой градиент и облегчение начертания логотипа Errorium в хедере лендинга (Milestone 1, Iteration 39)

### Затронутые компоненты и спецификации:
- `services/frontend/static/styles.css`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Стилизация логотипа в хедере (.landing-header-logo):**
   - В [styles.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/styles.css) цвет логотипа изменен с чисто белого на фирменный лазурно-голубой градиент (`linear-gradient(135deg, #60a5fa 0%, #38bdf8 50%, #93c5fd 100%)`).
   - Начертание шрифта облегчено с массивного `font-weight: 800` до более утонченного `font-weight: 600 (Semi-bold)`.
2. **Верификация и тесты:**
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).

## [2026-11-16] — Удаление рамки маскота в хедере и его горизонтальный разворот вправо (Milestone 1, Iteration 38)

### Затронутые компоненты и спецификации:
- `services/frontend/static/index.html`
- `services/frontend/static/styles.css`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Чистый развернутый маскот в хедере лендинга:**
   - В [index.html](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/index.html) удалена круглая рамка и фон. Маскот теперь представлен чистым элементом `.landing-header-mascot-img`.
   - В [styles.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/styles.css) применен `transform: scaleX(-1)`: маскот зеркально развернут и теперь естественно смотрит вправо (в сторону кнопок навигации и контента).
   - На ховер добавлен аккуратный микро-зум и фирменная синяя тень (`drop-shadow`).
2. **Верификация и тесты:**
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).

## [2026-11-16] — Установка размера 0.95rem для кнопок навигации в хедере лендинга (Milestone 1, Iteration 37)

### Затронутые компоненты и спецификации:
- `services/frontend/static/styles.css`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Калибровка шрифта кнопок хедера (.landing-anchor-link):**
   - В [styles.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/styles.css) для кнопок `MCP` и `About us` размер шрифта установлен в ровно `0.95rem` (~`15.2px`).
2. **Верификация и тесты:**
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).

## [2026-11-16] — Добавление круглого SVG-маскота Errorium в левый край хедера лендинга (Milestone 1, Iteration 36)

### Затронутые компоненты и спецификации:
- `services/frontend/static/index.html`
- `services/frontend/static/styles.css`
- `services/frontend/static/app.js`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Круглый маскот в шапке лендинга (.landing-mascot-circle):**
   - В [index.html](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/index.html) в левый угол хедера перед якорными ссылками `MCP` и `About us` добавлен круглый интерактивный бейдж с маскотом Errorium.
   - Бейдж реализован в виде векторного круглого SVG (`<svg class="mascot-round-svg">`) с клиппингом (`clipPath`), мягким фоновым радиальным градиентом `#1e293b -> #0f172a` и тонким лазурным ободком.
   - В [styles.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/styles.css) добавлен микро-ховер: плавное увеличение `scale(1.08)` и мягкое неоновое свечение.
   - В [app.js](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/app.js) клик по маскоту плавно возвращает пользователя к началу страницы лендинга.
2. **Верификация и тесты:**
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).

## [2026-11-16] — Установка точного размера 0.9rem для якорных кнопок хедера лендинга (Milestone 1, Iteration 35)

### Затронутые компоненты и спецификации:
- `services/frontend/static/styles.css`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Калибровка шрифта кнопок хедера (.landing-anchor-link):**
   - В [styles.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/styles.css) для кнопок `MCP` и `About us` размер шрифта установлен в ровно `0.9rem` (аккуратный баланс между читаемостью и компактностью).
   - Отступы скорректированы до `7px 16px`.
2. **Верификация и тесты:**
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).

## [2026-11-16] — Увеличение размера шрифта элементов в хедере лендинга (Milestone 1, Iteration 34)

### Затронутые компоненты и спецификации:
- `services/frontend/static/styles.css`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Крупная типографика хедера лендинга:**
   - В [styles.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/styles.css) для якорных кнопок навигации `.landing-anchor-link` (`MCP`, `About us`) размер шрифта увеличен с `0.88rem` до **`1.05rem`**, внутренние отступы увеличены до `8px 18px`.
   - Для логотипа `#landing-header-logo` (`Errorium`) в правом углу размер шрифта увеличен с `1.55rem` до **`1.75rem`**, начертание усилено до `font-weight: 800`.
   - Вся шапка стала солиднее, выразительнее и лучше читается на десктопных и мобильных экранах.
2. **Верификация и тесты:**
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).

## [2026-11-16] — Упрощение якорей лендинга до чистых #mcp и #about (Milestone 1, Iteration 33)

### Затронутые компоненты и спецификации:
- `services/frontend/static/index.html`
- `services/frontend/static/app.js`
- `tests/unit/test_frontend_spa.py`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Чистые названия якорей:**
   - В [index.html](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/index.html) удалён лишний якорь на начало страницы (Overview).
   - В хедере оставлены ровно 2 аккуратные якорные ссылки: `href="#mcp"` и `href="#about"`.
   - Сами секции получили лаконичные id: `<section id="mcp">` и `<section id="about">`.
   - В [app.js](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/app.js) контроллер `switchToLandingView` теперь использует чистые якоря `'mcp'` и `'about'`, а при переходе наверх страницы (по клику на логотип в сайдбаре) просто скроллит страницу к началу без лишних хэшей.
2. **Верификация и тесты:**
   - Тест `test_landing_page_structure_and_anchors` актуализирован.
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).

## [2026-11-16] — Фиксация единого URL лендинга с якорной навигацией без смены путей страниц (Milestone 1, Iteration 32)

### Затронутые компоненты и спецификации:
- `services/frontend/static/app.js`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Якорная навигация внутри единой страницы лендинга:**
   - В [app.js](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/app.js) в функции `switchToLandingView()` путь страницы больше не перезаписывается на отдельные страницы (`/about` или `/mcp`).
   - Лендинг сохраняет базовый путь страницы (`/landing` или `/overview`) и переключает исключительно хэш-якоря (`#landing-intro`, `#landing-mcp`, `#landing-about`).
   - Навигация по разделам происходит бесшовно через естественные якори и плавный скролл.
2. **Верификация и тесты:**
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).

## [2026-11-16] — Возврат межстрочного интервала 1.8 для вводного описания лендинга (Milestone 1, Iteration 31)

### Затронутые компоненты и спецификации:
- `services/frontend/static/styles.css`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Восстановление межстрочного интервала (.landing-intro-description):**
   - В [styles.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/styles.css) значение `line-height` возвращено к комфортному просторному `1.8`.
2. **Верификация и тесты:**
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).

## [2026-11-16] — Добавление прямых URL-маршрутов /landing и /overview для лендинга Errorium (Milestone 1, Iteration 30)

### Затронутые компоненты и спецификации:
- `services/frontend/static/app.js`
- `tests/unit/test_frontend_spa.py`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **SPA-роутинг для лендинга (/landing и /overview):**
   - В [app.js](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/app.js) в функцию `renderCurrentRoute()` добавлена поддержка маршрутов `path === '/landing'` и `path === '/overview'`. При переходе по этим URL открывается лендинг с автоматической фокусировкой на вводном Hero-разделе `#landing-intro`.
   - В функции `switchToLandingView()` обновлена логика синхронизации URL: при переходе к вводному разделу в адресной строке выставляется `/landing` (или сохраняется `/overview`), для MCP — `/mcp`, для About — `/about`.
2. **Верификация и тесты:**
   - Обновлен тест `test_landing_page_structure_and_anchors` в `tests/unit/test_frontend_spa.py`.
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).

## [2026-11-16] — Уменьшение отступа между заголовком Hello и описанием проекта (Milestone 1, Iteration 29)

### Затронутые компоненты и спецификации:
- `services/frontend/static/styles.css`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Верхний отступ описания (.landing-intro-description):**
   - В [styles.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/styles.css) для `.landing-intro-description` задан `margin: -8px 0 0 0`, сокративший промежуток после крупного заголовка `Hello, this is Errorium.`.
   - Заголовок и вводный абзац теперь визуально воспринимаются как единая цельная текстовая группа.
2. **Верификация и тесты:**
   - 24/24 теста пройдены (`python3 -m unittest discover tests/unit`).

## [2026-11-16] — Корректировка межстрочного интервала описания лендинга до 1.6 (Milestone 1, Iteration 28)

### Затронутые компоненты и спецификации:
- `services/frontend/static/styles.css`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Межстрочный интервал (.landing-intro-description):**
   - В [styles.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/styles.css) параметр `line-height` уменьшен с `1.8` до деликатного `1.6`.
   - Текст стал компактнее и сбитнее, сохранив при этом отличную читаемость моноширинных символов.
2. **Верификация и тесты:**
   - 24/24 теста пройдены (`python3 -m unittest discover tests/unit`).

## [2026-11-16] — Перемещение описания проекта сразу под заголовок Hello, this is Errorium (Milestone 1, Iteration 27)

### Затронутые компоненты и спецификации:
- `services/frontend/static/index.html`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Перестановка блоков на первом экране:**
   - В [index.html](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/index.html) параграф с описанием `.landing-intro-description` (шрифт JetBrains Mono) теперь идёт **сразу же под заголовком `Hello, this is Errorium.`**.
   - Карточка с хим. элементом `Element 404` перемещена ниже описания — непосредственно перед блоком кнопок запуска.
   - Логика восприятия стала естественной: приветствие -> суть и назначение системы -> карточка концепта Element 404 -> кнопки перехода к отладке.
2. **Верификация и тесты:**
   - 24/24 теста пройдены (`python3 -m unittest discover tests/unit`).

## [2026-11-16] — Увеличение размера шрифта JetBrains Mono для вводного описания лендинга (Milestone 1, Iteration 26)

### Затронутые компоненты и спецификации:
- `services/frontend/static/styles.css`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Увеличение шрифта JetBrains Mono (.landing-intro-description):**
   - В [styles.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/styles.css) размер шрифта моноширинного описания увеличен с `0.95rem` до `1.08rem` (~`17.3px`).
   - Скорректирован межстрочный интервал до `1.8` и максимальная ширина контейнера до `860px` для максимального удобства чтения текста.
2. **Верификация и тесты:**
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).

## [2026-11-16] — Увеличение размера и визуального веса главной кнопки «Start debugging with Errorium» (Milestone 1, Iteration 25)

### Затронутые компоненты и спецификации:
- `services/frontend/static/styles.css`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Увеличение кнопки действия (.landing-hero-cta-btn):**
   - В [styles.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/styles.css) увеличены внутренние отступы с `12px 24px` до `16px 32px` и радиус скругления до `14px`.
   - Размер шрифта увеличен с `0.96rem` до `1.08rem`, начертание усилено до `font-weight: 700`.
   - Добавлена объемная неоновая тень (`box-shadow: 0 8px 28px rgba(37, 87, 232, 0.45), 0 0 20px rgba(56, 189, 248, 0.2)`).
   - При ховере реализована плавная микротрансформация `scale(1.02)` и динамический сдвиг стрелочки на 3px вправо.
2. **Верификация и тесты:**
   - 24/24 теста успешно пройдены (`python3 -m unittest discover tests/unit`).

## [2026-11-16] — Перевод шрифта описания поисковика на JetBrains Mono (Milestone 1, Iteration 24)

### Затронутые компоненты и спецификации:
- `services/frontend/static/styles.css`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Шрифт вступительного описания (.landing-intro-description):**
   - В [styles.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/styles.css) для описания задан хакерский моноширинный шрифт `font-family: var(--font-mono)` (`JetBrains Mono`).
   - Скорректированы параметры верстки: `font-size: 0.95rem`, `line-height: 1.75`, `letter-spacing: -0.015em`, `color: #cbd5e1`.
   - Текст идеально сочетается с терминальной инженерной тематикой отладки ошибок и аварийных дампов.
2. **Верификация и тесты:**
   - Модульные тесты 24/24 пройдены (`python3 -m unittest discover tests/unit`).

## [2026-11-16] — Удаление блока плашек с техническими метриками (Zero third-party, Sub-millisecond, Cursor/Claude) (Milestone 1, Iteration 23)

### Затронутые компоненты и спецификации:
- `services/frontend/static/index.html`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Разгрузка вступительного Hero-экрана лендинга:**
   - В [index.html](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/index.html) удалён блок `.landing-hero-pills-row`, содержавший 3 плашки (`Zero third-party search engines`, `Sub-millisecond gRPC search core`, `Cursor & Claude Code integration`).
   - Страница стала чище и лаконичнее, без избыточного дублирования технических тезисов перед кнопками действий.
2. **Верификация и тесты:**
   - 24/24 модульных теста пройдены успешно (`python3 -m unittest discover tests/unit`).

## [2026-11-16] — Осветление шоукейса Element 404 и удаление надписи 100 GB (Milestone 1, Iteration 22)

### Затронутые компоненты и спецификации:
- `services/frontend/static/index.html`
- `services/frontend/static/styles.css`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Удаление подписи «100 GB»:**
   - В [index.html](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/index.html) удалена строка `<span class="element-mass">100 GB</span>`, плитка элемента стала более чистой и сфокусированной на `404`, `Er` и `Errorium`.
2. **Осветление и контраст блока:**
   - В [styles.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/styles.css) фон карточки осветлен с глухого темного `#0d131f` до благородного сланцево-синего `#172033` с рамкой `#2d3b55`.
   - Плитка элемента получила цвет `#1e293b` с четким контуром `#475569`.
   - Текст символа `Er` и слогана стал ярче и контрастнее (`#ffffff` и `#f8fafc`), а бейдж `Periodic Table of Engineering` приобрел лазурный акцент (`#38bdf8`).
3. **Верификация и тесты:**
   - 24/24 теста пройдены (`python3 -m unittest discover tests/unit`).

## [2026-11-16] — Перевод шоукейса Element 404 в плотный матовый технический стиль (Milestone 1, Iteration 21)

### Затронутые компоненты и спецификации:
- `services/frontend/static/styles.css`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Смена стиля карточки Element 404 (Плотный матовый инжиниринг):**
   - Убрано прозрачное стекло (`backdrop-filter`) и неоновое свечение.
   - Задан глубокий матовый фон `#0d131f` со строгой тонкой рамкой `#1e293b`.
   - Плитка элемента выполнена в виде плотного темного блока `#111827` со строгой окантовкой `#374151` и чётким белым символом `Er` (`#f8fafc`).
   - Блок теперь выглядит основательно, сдержанно и технично, создавая контраст с более мягкими элементами страницы.
2. **Верификация и тесты:**
   - 24/24 теста пройдены (`python3 -m unittest discover tests/unit`).

## [2026-11-16] — Фирменный редизайн слогана в стиле карточки элемента таблицы Менделеева (Milestone 1, Iteration 20)

### Затронутые компоненты и спецификации:
- `services/frontend/static/index.html`
- `services/frontend/static/styles.css`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Визуализация химического элемента 404 (Element 404 showcase):**
   - В [index.html](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/index.html) и [styles.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/styles.css) создан интерактивный шоукейс `.landing-element-showcase`:
     - Мини-плитка элемента `.element-tile-card` в стиле таблицы Менделеева:
       - Атомный номер: `404` (акцентный коралловый цвет `#ef4444`);
       - Символ: крупный градиентный `Er` с неоновым лазурным свечением;
       - Имя элемента: `Errorium`;
       - Атомная масса: объем дампа `100 GB`.
     - Справа от плитки размещен органичный блок слогана:
       - Неоновый бейдж: `Periodic Table of Engineering`;
       - Заголовок: `Element 404. Reacts with developers.`;
       - Подстрочник: `Stack overflow. Issue overknown.` с подсветкой ключевой фразы.
   - Шоукейс оформлен мягким стеклянным блюром (`backdrop-filter: blur(12px)`) и аккуратным лазурным ховером, идеально вписываясь в стилистику поисковика.
2. **Верификация и тесты:**
   - Модульные тесты пройдены: 24/24 OK (`python3 -m unittest discover tests/unit`).

## [2026-11-16] — Переход к чистому премиальному Smooth Float появлению логотипа в хедере (Milestone 1, Iteration 19)

### Затронутые компоненты и спецификации:
- `services/frontend/static/styles.css`
- `services/frontend/static/app.js`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Чистый кинематографичный Smooth Float (стиль Apple / Vercel):**
   - Полностью убран весь визуальный шум: сжатия слова, дырки в тексте, летящие точки, искры и искусственные автоскроллы.
   - Текст `Hello, this is Errorium.` на странице остаётся чистым, статичным и красивым презентационным заголовком.
   - При естественном пролистывании страницы, как только блок заголовка скрывается за верхней линией под хедером, в правом углу хедера плавно, мягко и кинематографично проявляется логотип `Errorium` (`translateY(6px) -> translateY(0)`, `blur(4px) -> blur(0)`, `opacity: 0 -> 1` за 0.32s cubic-bezier).
   - При скролле обратно наверх логотип так же мягко растворяется.
2. **Верификация и тесты:**
   - Модульные тесты пройдены: 24/24 OK (`python3 -m unittest discover tests/unit`).

## [2026-11-16] — Эффект Particle Disperse: распад слова Errorium в искры и их сбор в синий огонёк (Milestone 1, Iteration 18)

### Затронутые компоненты и спецификации:
- `services/frontend/static/styles.css`
- `services/frontend/static/app.js`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Анимация Particle Disperse:**
   - Вместо монолитного сжатия слово `Errorium` при начале скролла мгновенно распадается на светящийся рой из 18 неоновых микрочастиц (`.landing-disperse-sparkle`) с мягким размытием исходного текста.
   - Искры по динамическим сходящимся траекториям слетаются в правую световую точку, за 280мс зажигая яркий синий огонёк.
   - Огонёк со шлейфом искр летит по диагонали в правый верхний угол хедера и там разрастается в заголовок `Errorium`.
2. **Верификация и тесты:**
   - 24/24 модульных теста пройдены (`python3 -m unittest discover tests/unit`).

## [2026-11-16] — Редизайн блока слогана: удаление тяжелой рамки и кода в пользу чистого типографического стиля (Milestone 1, Iteration 17)

### Затронутые компоненты и спецификации:
- `services/frontend/static/styles.css`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Визуальное обновление блока слогана:**
   - В [styles.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/styles.css) полностью удалена рамка, серая подложка и левая синяя полоска (`border: none; background: transparent; padding: 2px 0 6px 0;`).
   - Убран моноширинный технический вид в пользу чистого, элегантного заголовка продукта (`font-family: inherit; font-size: 1.25rem - 1.6rem; color: #f1f5f9; font-weight: 600`).
   - Подстрочник `Stack overflow. Issue overknown.` гармонично дополняет слоган мягким матовым оттенком (`#94a3b8`).
2. **Верификация и тесты:**
   - 24/24 модульных теста успешно пройдены (`python3 -m unittest discover tests/unit`).

## [2026-11-16] — Плавное исчезновение точки в конце предложения при сжатии слова Errorium (Milestone 1, Iteration 16)

### Затронутые компоненты и спецификации:
- `services/frontend/static/index.html`
- `services/frontend/static/styles.css`
- `services/frontend/static/app.js`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Синхронное угасание точки после слова:**
   - В [index.html](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/index.html) точка в конце предложения вынесена в отдельный спан `#landing-welcome-dot`.
   - В [styles.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/styles.css) добавлен класс `.welcome-title-dot.fade-out` с плавным переходом `opacity: 0` и `transform: scale(0.4)`.
   - В [app.js](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/app.js) при старте сжатия слова точка мягко тает одновременно с превращением слова в огонёк. При возврате наверх точка плавно восстанавливает видимость.
2. **Верификация и тесты:**
   - Модульные тесты 24/24 пройдены (`python3 -m unittest discover tests/unit`).

## [2026-11-16] — Плавный автоскролл-доводчик при сжатии слова Errorium (Milestone 1, Iteration 15)

### Затронутые компоненты и спецификации:
- `services/frontend/static/app.js`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Интеллектуальный доводчик скролла (Auto-scroll closer):**
   - В момент, когда пользователь инициирует скролл и слово начинает сжиматься в точку, автоматически подключается плавный доводчик: `scrollBody.scrollTo({ top: targetScrollTop, behavior: 'smooth' })`.
   - Целевая координата рассчитывается так, чтобы заголовок `Hello, this is Errorium` мягко и синхронно уплыл за верхний край хедера в точности во время полёта огонька, благодаря чему пользователь никогда не видит статичной строки без слова.
2. **Верификация и тесты:**
   - Модульные тесты пройдены: 24/24 OK (`python3 -m unittest discover tests/unit`).

## [2026-11-16] — Мгновенный запуск сжатия Errorium в точку при самом первом скролле вниз (Milestone 1, Iteration 14)

### Затронутые компоненты и спецификации:
- `services/frontend/static/app.js`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Мгновенная реакция на скролл:**
   - В `initLandingLogoScrollAnimation` условие срабатывания переведено на `scrollTop > 8px`. Теперь не нужно ждать, пока блок заголовка доедет до хедера: как только пользователь делает первое движение колесом мыши или тач-скролл вниз, слово `Errorium` в фразе моментально схлопывается вправо в точку, превращается в синий огонёк и начинает свой полет в хедер.
   - При скролле на самый верх (`scrollTop <= 4px`) слово плавно разворачивается обратно.
2. **Верификация и тесты:**
   - 24/24 теста пройдены (`python3 -m unittest discover tests/unit`).

## [2026-11-16] — Синхронная анимация сжатия слова Errorium вправо непосредственно внутри фразы заголовка (Milestone 1, Iteration 13)

### Затронутые компоненты и спецификации:
- `services/frontend/static/styles.css`
- `services/frontend/static/app.js`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Сжатие слова Errorium непосредственно в тексте фразы:**
   - Для элемента `#landing-welcome-brand` в фразе `Hello, this is Errorium.` задан `display: inline-block` и точка трансформации `transform-origin: right center`.
   - При скролле класс `.in-place-collapsed` запускает плавное аппаратное сжатие самого слова вправо (`scaleX(0.01) scaleY(0.08)`) одновременно со сжатием клона.
   - Слово схлопывается в точку прямо в строке заголовка, из которой сразу материализуется синий огонёк с искрами и устремляется в хедер.
   - При пролистывании страницы обратно вверх класс удаляется, и слово плавно восстанавливает свой исходный размер.
2. **Верификация и тесты:**
   - Синтаксис проверен (`node --check`).
   - Модульные тесты 24/24 пройдены (`python3 -m unittest discover tests/unit`).

## [2026-11-16] — Стилизация летящей точки под синий огонёк Errorium с мерцанием и искрами (Milestone 1, Iteration 12)

### Затронутые компоненты и спецификации:
- `services/frontend/static/styles.css`
- `services/frontend/static/app.js`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Эффект синего огонька (Blue Fire Flame & Embers):**
   - Точка `.landing-flame-dot` стилизована под синий язык пламени (в точном соответствии с фирменной аурой `blue-fire.png` чата):
     - Каплевидная форма пламени (`border-radius: 50% 50% 40% 40% / 60% 60% 40% 40%`);
     - Многослойный градиент ядра: от белой раскалённой сердцевины (`#ffffff`) к лазурному (`#38bdf8`) и глубокому индиго (`#1d4ed8`);
     - Трёхуровневое радиальное неоновое свечение (box-shadow) и ореол пламени (`::before`);
     - Живая микроанимация мерцания язычка `@keyframes flameFlicker` (микропокачивания и колебания яркости);
     - Генерация микроискр `.landing-flame-spark` по траектории полёта, плавно рассеивающихся позади огонька.
2. **Верификация и тесты:**
   - 24/24 модульных теста пройдены (`python3 -m unittest discover tests/unit`).

## [2026-11-16] — Смещение надписи Errorium в хедере лендинга в крайний правый упор (Milestone 1, Iteration 11)

### Затронутые компоненты и спецификации:
- `services/frontend/static/styles.css`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Полноширинный хедер лендинга:**
   - В [styles.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/styles.css) убран лимит `max-width: 1200px` у `.landing-header-inner` (заменен на `100%`), правый отступ уменьшен до минимального (`padding: 0 16px 0 24px`), а для `#landing-header-logo` задан `margin-left: auto`.
   - В результате логотип `Errorium` теперь прижат до упора к правому краю окна браузера, а диагональный полёт точки динамически рассчитывает координаты до крайнего правого положения.
2. **Верификация и тесты:**
   - 24/24 модульных теста пройдены (`python3 -m unittest discover tests/unit`).

## [2026-11-16] — Диагональный полёт точки вправо-вверх и прижатие логотипа к правому краю хедера лендинга (Milestone 1, Iteration 10)

### Затронутые компоненты и спецификации:
- `services/frontend/static/index.html`
- `services/frontend/static/styles.css`
- `services/frontend/static/app.js`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Позиционирование логотипа в хедере:**
   - Логотип `Errorium` (`#landing-header-logo`) прижат к правому краю внутренней области хедера (`.landing-header-inner` с flex `justify-content: space-between`).
2. **Диагональная траектория полета точки:**
   - После схлопывания надписи вправо в светящуюся точку `.landing-morph-dot`, траектория полёта направлена по диагонали вправо-вверх к правой части хедера.
   - По прибытии в правый край хедера точка вспыхивает, и логотип разрастается от правого края (`transform-origin: right center`).
3. **Верификация и тесты:**
   - Синтаксис `node --check` проверен.
   - Модульные тесты 24/24 OK (`python3 -m unittest discover tests/unit`).

## [2026-11-16] — Эффектная анимация сжатия надписи Errorium в точку и её разрастания в хедере (Milestone 1, Iteration 9)

### Затронутые компоненты и спецификации:
- `services/frontend/static/styles.css`
- `services/frontend/static/app.js`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Многофазная физическая анимация перехода логотипа (Morph & Flight):**
   - **Фаза 1 (Сжатие вправо):** При скрытии `Hello, this is Errorium` за верхнюю границу экрана текст клонируется с точкой трансформации `transform-origin: right center` и сжимается вправо (`scaleX(0.01) scaleY(0.08)`), схлопываясь в крайнюю правую координату.
   - **Фаза 2 (Светящаяся точка-квант):** В точке схлопывания материализуется светящаяся неоновая точка `.landing-morph-dot` с мягким неоновым шлейфом (`box-shadow: 0 0 14px #38bdf8`).
   - **Фаза 3 (Подъём в хедер):** Точка плавно поднимается по кривой Безье в центр верхнего хедера (к месту логотипа).
   - **Фаза 4 (Вспышка и разрастание):** В центре хедера точка вспыхивает и рассеивается, а надпись `#landing-header-logo` эффектно разрастается из центра (`scale(0.15) -> scale(1.08) -> scale(1.0)`) с неоновым свечением.
   - При скролле обратно наверх хедер аккуратно скрывает логотип, подготавливая его к повторной анимации.
2. **Верификация и тесты:**
   - Синтаксис проверен (`node --check`).
   - Все 24 модульных теста пройдены (`python3 -m unittest discover tests/unit`).

## [2026-11-16] — Увеличение верхнего отступа лендинга от хедера до начала контента (Milestone 1, Iteration 8)

### Затронутые компоненты и спецификации:
- `services/frontend/static/styles.css`
- `CHANGE_HISTORY.md`
- `STATUS.md`

### Суть изменений:
1. **Отступ от хедера до контента:**
   - В файле [styles.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/styles.css) увеличен верхний отступ контейнера скролла `.landing-scroll-body` с `40px` до `72px`, а также внутренний верхний отступ вводного блока `.landing-intro-section` с `10px` до `24px` (суммарный отступ от фиксированного хедера теперь составляет около `96px`).
   - Скорректирован `scroll-margin-top` секций до `36px` для гармоничного скролла по якорным ссылкам.
2. **Верификация и тесты:**
   - Тесты пройдены: 24/24 OK (`python3 -m unittest discover tests/unit`).

## [2026-11-16] — Добавление эффектного вступительного Hero-блока на лендинге с приветствием и слоганом (Milestone 1, Iteration 6)

### Затронутые компоненты и спецификации:
- `docs/Product/04-interface-and-behavior.md`
- `services/frontend/static/index.html`
- `services/frontend/static/styles.css`
- `services/frontend/static/app.js`
- `tests/unit/test_frontend_spa.py`
- `STATUS.md`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Вступительный Hero-раздел лендинга (`#landing-intro`):**
   - Лендинг теперь начинается не с технических деталей MCP, а с эффектного презентационного экрана:
     - Неоновый бейдж: `Next-Gen Engineering Search`;
     - Крупный заголовок с градиентным свечением: `Hello, this is Errorium.`;
     - Фирменная карточка слогана в моноширинном стиле: `Element 404. Reacts with developers` / `Stack overflow. Issue overknown.`;
     - Вводное описание назначения поисковика и RAG-синтеза для программных ошибок и аварийных дампов;
     - Бейджи ключевых достоинств: Zero third-party search engines, Sub-millisecond gRPC search core, Cursor & Claude Code integration;
     - Главные интерактивные кнопки: `Start debugging with Errorium` (переход в чат с фокусом на ввод) и `Explore MCP & Architecture` (плавный скролл к техническим блокам).
2. **Якорная навигация:**
   - В разделительную полосу добавлен якорный переключатель `Overview` (`#landing-intro`) перед `MCP` и `About us`.
3. **Верификация и тесты:**
   - Модульные тесты обновлены и успешно пройдены: 24/24 OK.

## [2026-11-16] — Вынос лендинга в изолированный полноэкранный интерфейс с кнопкой Let's try и сквозным хедером (Milestone 1, Iteration 5)

### Затронутые компоненты и спецификации:
- `docs/Product/04-interface-and-behavior.md`
- `services/frontend/static/index.html`
- `services/frontend/static/styles.css`
- `services/frontend/static/app.js`
- `tests/unit/test_frontend_spa.py`
- `STATUS.md`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Изолированный полноэкранный лендинг (`.landing-standalone-page`):**
   - При переходе на лендинг через кнопки сайдбара (`MCP`, `About us`) или по URL (`/mcp`, `/about`) рабочая область чата и сайдбар полностью скрываются (`body.landing-active`), открывая страницу с нуля.
   - В шапке лендинга по центру размещен фирменный логотип `Errorium` (как в основном приложении), слева кнопка возврата `Back to Chat`, а справа акцентная градиентная кнопка `Let's try`, возвращающая пользователя в чат и фокусирующая ввод.
   - Разделительная полоса навигации (`.landing-divider-bar`) растянута на всю ширину экрана (`100vw`) с переключателями секций `MCP` и `About us`.
2. **Верификация и тесты:**
   - Обновлен тест `test_landing_page_structure_and_anchors` в `tests/unit/test_frontend_spa.py`.
   - Проверка синтаксиса `node --check`: чисто.
   - Модульные тесты: 24/24 успешно (OK).

## [2026-11-16] — Реализация информационного лендинга Errorium с якорными переходами к MCP и About us (Milestone 1, Iteration 4)

### Затронутые компоненты и спецификации:
- `docs/Product/04-interface-and-behavior.md`
- `services/frontend/static/index.html`
- `services/frontend/static/styles.css`
- `services/frontend/static/app.js`
- `tests/unit/test_frontend_spa.py`
- `STATUS.md`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Разработка отдельной страницы лендинга (`#landing-view`):**
   - Главный экран чата (`#hero-view`) полностью возвращен к оригинальному минималистичному дизайну (парящий маскот, синий огонь, точный слоган `Element 404...`, капсульная строка ввода).
   - Лендинг вынесен в отдельный самостоятельный экран `#landing-view`, который активируется при нажатии на `MCP` или `About us` в боковой панели, а также по маршрутам `/mcp` и `/about`.
   - В верхней части экрана лендинга предусмотрены кнопка возврата в чат (`Back to Chat`) и переключатели якорей (`MCP` и `About us`).
   - При переходе на лендинг нижний prompt bar скрывается, чтобы не заслонять контент. При возврате в чат (кнопка `Back to Chat` или `New chat`) восстанавливается исходный экран.
   - **Секция `#landing-mcp`:** карточки преимуществ (Remote MCP over SSE, точная токенизация стектрейсов для BM25, защищенный Bearer токен), интерактивный блок с персональным токеном и JSON-конфигурацией для `~/.cursor/mcp.json` и `claude_desktop_config.json` с кнопками копирования.
   - **Секция `#landing-about`:** академическая спецификация НИРС ИУ5 МГТУ им. Н. Э. Баумана (100 GB дамп SO, BM25 с нуля $k_1=1.5, b=0.75$, sub-millisecond IPC по gRPC, потоковый RAG), карточка философии системы и метрик ранжирования.
2. **Якорная навигация и роутинг:**
   - Функция `switchToLandingView(updateRoute, targetAnchor)` выполняет переключение экранов и плавный скролл к выбранному якорю.
3. **Локализация:**
   - Добавлены двусторонние переводы (RU <-> EN) для всех заголовков, бейджей и действий лендинга.
4. **Тестирование и верификация:**
   - Добавлен модульный тест `test_landing_page_structure_and_anchors` в `tests/unit/test_frontend_spa.py`.
   - Синтаксис проверен через `node --check services/frontend/static/app.js` (код 0).
   - Все 24 unit-теста пройдены успешно: `python3 -m unittest discover tests/unit` (OK).

## [2026-11-14] — Восстановление исходного дизайна сообщений чата в Web Frontend (Milestone 1, Iteration 3)

### Затронутые компоненты и спецификации:
- `services/frontend/static/app.js`
- `services/frontend/static/styles.css`
- `tests/unit/test_frontend_spa.py`
- `STATUS.md`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Восстановление внешнего вида и верстки сообщений (R1):**
   - Пользовательские сообщения возвращены к `.user-bubble` (темный фон `#1a233b`, граница `1px solid rgba(255, 255, 255, 0.12)`, скругление `18px 18px 4px 18px`, моноширинный шрифт `var(--font-mono)`, белый цвет `#ffffff`) без иконки галочки доставки.
   - Ответы ассистента теперь напрямую отображают карточку `.assistant-card` (Errorium RAG Synthesis BM25 Core) без вспомогательного пузыря приветствия и без обертки с аватаром `.dialog-assistant-wrapper`.
   - Контейнер `.messages-container` сохраняет строки `.message-row.user` и `.message-row.assistant`.
   - Из `styles.css` удалены неиспользуемые стили `.dialog-*` и переменные `--color-user-bubble`, `--color-bot-bubble`.
2. **Обновление набора тестов (R2):**
   - Тест `test_chat_message_appearance_and_layout` проверяет корректность стилей `.user-bubble`, прямого рендеринга `.assistant-card` и отсутствие устаревших классов диалоговых пузырей и иконок.
   - Обеспечена обратная совместимость вызова теста.
3. **Результаты тестирования:**
   - Модульные тесты: `python3 -m unittest tests/unit/test_frontend_spa.py` (18 тестов успешно, 0 ошибок).
   - Полный запуск: `make test-unit` (21 unit-тест успешно).

## [2026-11-14] — Устранение синтаксической ошибки в app.js и добавление теста AST (Milestone 1, Iteration 2)

### Затронутые компоненты и спецификации:
- `services/frontend/static/app.js`
- `tests/unit/test_frontend_spa.py`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Устранение дублирующего фрагмента в `services/frontend/static/app.js`:**
   - Удалены строки с осиротевшим дублирующим фрагментом шаблонного литерала после функции `renderConversation`.
   - Синтаксис проверен через `node --check services/frontend/static/app.js` (код возврата 0, чистый синтаксис).
2. **Добавление регрессионного теста синтаксиса:**
   - В `tests/unit/test_frontend_spa.py` добавлен метод `test_app_js_syntax`, выполняющий проверку валидности JavaScript-кода через `node --check`.
3. **Результаты тестирования:**
   - Все 20 unit-тестов успешно пройдены: `python3 -m unittest discover -s tests/unit -p "test_*.py"` (20 tests passed in 0.62s).

## [2026-11-14] — Реализация недостающих экранов и компонентов интерфейса по макетам (Milestone 1)

### Затронутые компоненты и спецификации:
- `services/frontend/static/index.html`
- `services/frontend/static/styles.css`
- `services/frontend/static/app.js`
- `tests/unit/test_frontend_spa.py`
- `STATUS.md`
- `CHANGE_HISTORY.md`

### Суть изменений:
1. **Реализация референсных модальных окон (`services/frontend/references/`):**
   - **Экран авторизации (`Auth-screen.png`):** добавлен `#modal-auth-reference` с логотипом маскота, полями Email и Password со встроенными иконками, чекбоксом "Remember me", ссылкой "Forgot password?", зеленой пилюльной кнопкой "Log in" (`#2ebd64`), разделителем "or", кнопкой входа через Яндекс ID с красным бейджем "Я" и ссылкой "New here? Sign up".
   - **Запрос сброса пароля (`Change-password-email-screen.png`):** добавлен `#modal-change-password-email` с подзаголовком "Enter your account email", полем ввода почты с иконкой, зеленой кнопкой "Send reset link" и уведомлением о проверке папки со спамом.
   - **Смена пароля (`Change-password-screen.png`):** добавлен `#modal-change-password` с заголовком "Change password", полями ввода нового пароля и подтверждения, валидацией совпадения и зеленой кнопкой "Approve".
   - **Предупреждение о сбросе пароля (`Reset-password-alert.png`):** добавлен `#modal-reset-password-alert` с янтарной иконкой предупреждения, текстом предупреждения о выходе со всех устройств и янтарной кнопкой "Send reset link" (`#d99b43`).
2. **Экран профиля пользователя (`Profile-screen.png`):**
   - Реализован `#profile-view` с навигацией назад (`#btn-profile-back`), крупным фиолетовым аватаром с буквой "T" и бейджем редактирования с карандашом, заголовком имени пользователя `Test_user`.
   - Добавлены карточки персональной информации: имя пользователя (`👤 Test_user`), email (`✉ Test_user@mail.ru`), роль (`🎓 Backend developer`), интерактивный выбор пола (`⚥ Male` / `Female`).
   - Добавлен интерактивный блок безопасности ("Security" -> "Change password" с переходом к `#modal-reset-password-alert`).
   - Добавлен блок языка интерфейса ("Language" -> "English 文A").
3. **Поповер тегов технологий (`Chat-screen-with-tags.png`):**
   - Добавлен плавающий поповер `#tags-popover` с указателем-нотчем, привязанный к кнопке `+` строки ввода запроса.
   - Реализована двухрядная сетка из 8 референсных цветных тегов-пилюль: `Go` (`#1d72b8`), `Python` (`#c99700`), `Java` (`#b32424`), `C#` (`#7c3aed`), `C++` (`#16a34a`), `JS` (`#a855f7`), `SQL` (`#2563eb`), `Rust` (`#b45309`).
   - При клике на тег его значение в формате `[<Tag>] ` автоматически добавляется в строку ввода `#prompt-input`.
4. **Авторизованное состояние сайдбара и шапки (`Chat-screen-menu-authorized.png`):**
   - Добавлены переключаемые состояния сайдбара (`#sidebar-user-guest` и `#sidebar-user-auth`).
   - В авторизованном состоянии футер отображает фиолетовый круглый аватар "T", имя `Test_user`, статус `Backend developer` и иконку настроек.
   - В шапке кнопка профиля отображает фиолетовый аватар "T" вместо гостевой иконки; клик по аватару открывает `#profile-view`.
   - Состояние сессии сохраняется в `localStorage` по ключу `errorium_auth_user` (по умолчанию при первом запуске — гостевой режим).
5. **Диалоговые пузыри сообщений (`Chat-screen-dialog.png`):**
   - Добавлен стиль пользовательского сообщения в виде лавандово-синего пузыря `#6781db` с двойной галочкой доставки.
   - Добавлен стиль ответа ассистента с круглым аватаром маскота слева и серебристо-серым пузырем `#c0c1c6`.
   - Полностью сохранены карточка синтеза RAG BM25, блок исходного кода и список источников Stack Overflow.
6. **Строгое сохранение существующего интерфейса (R3):**
   - Полностью сохранены все существующие элементы `#hero-view`, гостевой сайдбар с 7 референсными ошибками, модальные окна MCP, About, Settings, десктопное сворачивание сайдбара и адаптивные медиа-запросы.
7. **Тестирование (19/19 тестов пройдено успешно):**
   - В `tests/unit/test_frontend_spa.py` добавлены 5 новых тестов: `test_reference_modals_structure`, `test_profile_screen_structure`, `test_tags_popover_structure`, `test_authorized_sidebar_and_header_state`, `test_conversational_dialog_bubbles`.
   - Все 19 тестов в `tests/unit/test_frontend_spa.py` и `tests/unit/test_docker_ports.py` успешно пройдены за 0.58 с. Чистый SPA без внешних фреймворков сохранен.

## [2026-10-10] — Миграция объектного хранилища S3 с MinIO на Garage

### Затронутые компоненты и спецификации:
- `deploy/garage/garage.toml`
- `docker-compose.yml`
- `docker-compose.dev.yml`
- `.env.example`
- `services/gateway/pkg/config/config.go`
- `services/gateway/pkg/config/config_test.go`
- `docs/Tech/02-running.md`
- `docs/Tech/03-data.md`
- `docs/Tech/04-api-and-access.md`
- `tests/unit/test_docker_ports.py`
- `STATUS.md`

### Суть изменений:
1. **Инфраструктура Garage S3 (`deploy/garage/garage.toml`, docker-compose):**
   - В связи с прекращением поддержки и удалением образов MinIO осуществлен переход на легковесное распределенное S3-совместимое хранилище **Garage** (образ `dxflrs/garage:v2.4.1`).
   - Добавлен базовый конфигурационный файл `deploy/garage/garage.toml` (sqlite движок метаданных, S3 API на порту 3900, Admin API на порту 3903, replication_factor = 1).
   - В `docker-compose.yml` и `docker-compose.dev.yml` сервис `minio` заменен на `garage` с флагами `--single-node --default-bucket`, автоматически создающими ноду, ключ доступа и бакет `errorium-avatars` при первом запуске.
   - Настроен healthcheck на основе нативной утилиты `/garage node id` (`test: ["CMD", "/garage", "node", "id"]`).
   - Тома обновлены на `errorium_garage_meta` и `errorium_garage_data` (и соответствующие dev-версии).
2. **Параметризация портов и `.env.example`:**
   - Введены переменные `GARAGE_PORT` (по умолчанию 3900) и `GARAGE_ADMIN_PORT` (по умолчанию 3903).
   - Обновлен `.env.example` с полным набором настроек Garage S3 (`GARAGE_DEFAULT_ACCESS_KEY`, `GARAGE_DEFAULT_SECRET_KEY`, `GARAGE_DEFAULT_BUCKET`, `GARAGE_REGION=garage`, `GARAGE_RPC_SECRET`, `GARAGE_ADMIN_TOKEN`).
3. **Конфигурация Gateway (`services/gateway/pkg/config`):**
   - Структура `MinIOConfig` генерализована в `S3Config` (с сохранением псевдонима `MinIOConfig = S3Config` и поля `cfg.MinIO` для обратной совместимости).
   - Добавлена поддержка региона S3 (`Region: "garage"`), а также чтение переменных окружения `S3_*`, `GARAGE_*` и fallback на `MINIO_*`.
   - Написаны модульные тесты конфигуратора в `services/gateway/pkg/config/config_test.go`.
4. **Тесты и спецификации:**
   - Обновлен тест `tests/unit/test_docker_ports.py` для валидации портов Garage S3.
   - В документации `docs/Tech/02-running.md`, `docs/Tech/03-data.md` и `docs/Tech/04-api-and-access.md` все упоминания MinIO скорректированы на Garage S3.
   - Все unit-тесты (Python и Go) пройдены успешно, конфигурации Docker Compose валидированы.

## [2026-10-09] — Параметризация портов сервисов для dev и prod окружений

### Затронутые компоненты и спецификации:
- `docker-compose.dev.yml`
- `docker-compose.yml`
- `.env.example`
- `docs/Tech/02-running.md`
- `tests/unit/test_docker_ports.py`
- `STATUS.md`

### Суть изменений:
1. **Параметризация портов в `docker-compose.dev.yml`:**
   - Для всех сервисов дев-окружения добавлен проброс портов через переменные окружения с дефолтными значениями:
     - `minio`: `127.0.0.1:${MINIO_PORT:-9000}:9000` и `127.0.0.1:${MINIO_CONSOLE_PORT:-9001}:9001` (позволяет сменить порт, если порт 9000 или 9001 занят на хост-машине).
     - `postgres`: `127.0.0.1:${POSTGRES_PORT:-5432}:5432`.
     - `gateway`: `127.0.0.1:${GATEWAY_PORT:-8000}:8000`.
     - `frontend`: `${FRONTEND_PORT:-3000}:80`.
   - Внутри Docker-сети сервисы продолжают взаимодействовать по фиксированным внутренним адресам (`minio:9000`, `postgres:5432`, `gateway:8000`), благодаря чему изменение портов хоста не ломает межсервисные связи.
2. **Параметризация портов в `docker-compose.yml`:**
   - Порты `gateway` (`127.0.0.1:${GATEWAY_PORT:-8000}:8000`) и `frontend` (`${FRONTEND_HTTP_PORT:-80}:80`, `${FRONTEND_HTTPS_PORT:-443}:443`) теперь также поддерживают переопределение через `.env`.
3. **Обновление `.env.example` и документации:**
   - В `.env.example` добавлены `POSTGRES_PORT=5432`, `MINIO_PORT=9000`, `MINIO_CONSOLE_PORT=9001` с поясняющими комментариями.
   - В `docs/Tech/02-running.md` дополнена сводная таблица портов и добавлена рекомендация по разрешению конфликтов портов на хосте.
4. **Тестирование:**
   - Добавлен модульный тест `tests/unit/test_docker_ports.py`, проверяющий корректность fallback-значений и регулярных выражений портов в конфигурациях compose и `.env.example`.
5. **Корректировка точки монтирования тома PostgreSQL:**
   - Путь монтирования тома PostgreSQL 18 в `docker-compose.dev.yml` и `docker-compose.yml` изменен с устаревшего `/var/lib/postgresql/data` на `/var/lib/postgresql` в соответствии с официальным образом `postgres:18-alpine`.

## [2026-10-09] — Проектирование схемы БД Gateway (PostgreSQL 18+), MinIO и миграций

### Затронутые компоненты и спецификации:
- `docs/Tech/03-data.md`
- `docs/Tech/04-api-and-access.md`
- `services/gateway/migrations/000001_init_gateway_schema.up.sql`
- `services/gateway/migrations/000001_init_gateway_schema.down.sql`
- `services/gateway/domain/` (`user.go`, `sso.go`, `session.go`, `chat.go`, `tag.go`, `audit.go`)
- `services/gateway/pkg/config/config.go`
- `docker-compose.yml`, `docker-compose.dev.yml`
- `.env.example`
- `STATUS.md`

### Суть изменений:
1. **Спецификация схемы PostgreSQL:**
   - Выбрана СУБД **PostgreSQL 18+**. Используется нативная встроенная функция ядра **`uuidv7()`** для первичных ключей (time-ordered идентификаторы без фрагментации B-Tree индексов).
   - Все строковые поля переведены с `VARCHAR(n)` на идиоматичный для PostgreSQL тип **`TEXT` с явными `CHECK`-ограничениями** длины (`length <= N`) и формата, что исключает накладные расходы на конвертацию и обеспечивает единый формат хранения `varlena`.

   - Сущности:
     - `users`: пользователи системы (роли `user` и `admin`, email, password_hash, avatar_url, статусы).
     - `user_sso_accounts`: OAuth/SSO профили (`google`, `yandex`, `github`), связь с пользователем, уникальность `(provider, provider_user_id)`.
     - `user_sessions`: сессии пользователей, SHA-256 хэши refresh-токенов, client_ip (INET), user_agent, статус отзыва (`is_revoked`), таймстемпы.
     - `chats`: темы диалогов пользователя (`Recent chats`), флаг закрепления `is_pinned`, быстрый композитный индекс `(user_id, updated_at DESC)` и триграммный поиск `pg_trgm`.
     - `chat_messages`: сообщения диалога, контекст ошибки `error_context` (JSONB), источники Stack Overflow `rag_sources` (JSONB), статистика токенов `token_usage` (JSONB).
     - `tags` и `chat_message_tags`: нормализованный каталог тегов и отношение Many-to-Many к сообщениям с каскадным удалением.
     - `audit_logs`: системный журнал критических действий.
   - **MCP:** Полностью исключена авторизация для MCP (удалены токены, эндпоинт `/sse` открыт без auth).
2. **Объектное хранилище аватарок (MinIO / S3):**
   - Добавлен сервис MinIO в `docker-compose.yml` и `docker-compose.dev.yml` для хранения аватарок пользователей (бакет `errorium-avatars`).
3. **Миграции и инфраструктура:**
   - Написаны файлы миграции `000001_init_gateway_schema.up.sql` и `000001_init_gateway_schema.down.sql`.
   - **Элегантные встроенные миграции (embed.FS):** SQL-файлы встроены в бинарник через `migrations.FS` (`embed.FS`). Добавлен раннер `internal/migrator` на базе `golang-migrate/v4/source/iofs`, который при старте сервиса Gateway атомарно накатывает ожидающие миграции.
   - **Очистка Docker:** Устранен антипаттерн монтирования одного SQL-файла в `/docker-entrypoint-initdb.d/`. Контейнер PostgreSQL в `docker-compose.yml` и `docker-compose.dev.yml` теперь полностью чист.
   - Созданы Go-модели в `services/gateway/domain/` и обновлен `config.go` для подключения к Postgres и MinIO (`AUTO_MIGRATE`).
   - Написан юнит-тест `services/gateway/migrations/migrations_test.go` для валидации встроенных миграций.

## [2026-10-08] — Подключение веб-шрифтов Inter и JetBrains Mono для кроссплатформенного отображения


### Затронутые компоненты и спецификации:
- `services/frontend/static/index.html`
- `services/frontend/static/styles.css`
- `STATUS.md`

### Суть изменений:
- В `<head>` [index.html](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/index.html) добавлены директивы `preconnect` к `fonts.googleapis.com` и `fonts.gstatic.com` с подключением шрифтов `Inter` (начертания 400, 500, 600, 700) и `JetBrains Mono` (начертания 400, 500, 600, 700).
- В [styles.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/styles.css) добавлен `@import` и обновлены системные переменные `--font-ui` и `--font-mono`.
- Теперь на мобильных устройствах (Android, iOS) и десктопах интерфейс и моноширинный слоган отображаются одинаково и аутентично макету.

## [2026-10-08] — Очистка и дедупликация тегов секции <head> в index.html

### Затронутые компоненты и спецификации:
- `services/frontend/static/index.html`
- `STATUS.md`

### Суть изменений:
- Устранены дублирующиеся теги: двойные `<meta charset="UTF-8">`, `<meta name="viewport">`, `<title>`, `<meta name="description">`, `<link rel="canonical">`, дублированные блоки Open Graph (`og:*`) и Twitter Card, повторные подключения фавиконов.
- Объединен расширенный список поисковых ключевых слов в единый тег `meta keywords` (`bm25, rag, mcp, java, python, go, c++`).
- Вычищены лишние пустые строки и сохранены все параметры адаптивного viewport (`viewport-fit=cover`), окрашивания системной шторки браузера (`#070a13`) и ранний анти-FOUC скрипт.

## [2026-10-08] — Отключение системного голубого выделения при нажатии на кнопки (Tap Highlight) на мобильных устройствах

### Затронутые компоненты и спецификации:
- `services/frontend/static/styles.css`
- `STATUS.md`

### Суть изменений:
- Для всех интерактивных элементов (`button`, `a`, `[role="button"]`, `.icon-btn`, а также глобально `*`) задано свойство `-webkit-tap-highlight-color: transparent !important;`.
- Отключен браузерный прямоугольный контур фокуса при таче (`outline: none !important`), а также системный вызов контекстного меню (`-webkit-touch-callout: none`) и выделение текста на кнопках (`user-select: none`).

## [2026-10-08] — Стилизация строки ввода сообщений с эффектом Liquid Glass

### Затронутые компоненты и спецификации:
- `services/frontend/static/styles.css`
- `docs/Product/04-interface-and-behavior.md`
- `STATUS.md`

### Суть изменений:
1. **Эффект Liquid Glass (Apple VisionOS / High-end UI Glass) для капсулы ввода (`.prompt-bar-container`):**
   - Переход от темного глухого фона к кристальной оптической прозрачности (`rgba(255, 255, 255, 0.12)` $\rightarrow$ `rgba(255, 255, 255, 0.03)` $\rightarrow$ `rgba(15, 23, 42, 0.35)`), позволяющей фону преломляться сквозь линзу.
   - Оптическое размытие подложки: `backdrop-filter: blur(28px) saturate(210%) brightness(108%)`.
   - Полное бесшовное заполнение капсулы градиентом стекла от края до края (удалены искусственные половинчатые псевдоэлементы `::before` / `::after`, создававшие эффект несоразмерного блика).
   - 5-слойный стек теней и внутренних светорассеяний (яркая кромка `inset 0 1.5px 1.5px rgba(255, 255, 255, 0.75)`, глубинное свечение и рассеянный амбиент).
2. **Элементы управления:**
   - Кнопка `+` (`.prompt-plus-btn`) сделана чистой и свободной от круглой плашки/фона — аккуратный отдельный символ с плавной сменой цвета и микро-анимацией масштабирования.
   - Кнопка отправки `.prompt-send-btn` стилизована под сапфировую каплю с 3D-куполом, двойным бликом и неоновым свечением.
   - Текст плейсхолдера и поле ввода адаптированы для высокой читаемости на стекле.

## [2026-10-08] — Устранение эффекта передергивания (FOUC) при перезагрузке со свернутым меню

### Затронутые компоненты и спецификации:
- `services/frontend/static/index.html`
- `services/frontend/static/styles.css`
- `services/frontend/static/app.js`
- `tests/unit/test_frontend_spa.py`
- `STATUS.md`

### Суть изменений:
1. **Ранняя синхронизация состояния в `<head>`:**
   - Добавлен компактный синхронный скрипт в `<head>` [index.html](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/index.html), который проверяет `localStorage` на сохраненное состояние `'collapsed'` до первого рендера DOM и навешивает класс `sidebar-collapsed` на `<html>`.
   - В [styles.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/styles.css) добавлены селекторы `html.sidebar-collapsed .sidebar-drawer` и шапки, чтобы браузер рендерил свернутое состояние с самого первого кадра без сдвига верстки (zero layout shift).
2. **Блокировка CSS-переходов на время начальной загрузки (`preload`):**
   - Добавлен класс `preload` на `<body>` и стили `.preload * { transition: none !important; }`.
   - В [app.js](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/app.js) класс `.preload` безопасно снимается после первой отрисовки через `requestAnimationFrame`, возвращая плавную анимацию для пользовательских кликов.
3. **Синхронизация в JS и фикс видимости кнопки разворачивания меню:**
   - Методы `collapseDesktopSidebar` и `expandDesktopSidebar` синхронно переключают класс на `document.documentElement`.
   - Кнопка разворачивания сайдбара в шапке `.header-sidebar-toggle-btn` (`#btn-expand-sidebar`) приведена к визуальному стилю мобильного гамбургер-меню (3 белые полоски через `.hamburger-bar`).
   - Исправлен конфликт каскада стилей, из-за которого кнопка открывания не исчезала при разворачивании меню, а съезжала в правую часть шапки: добавлено явное скрытие `display: none !important` при раскрытом сайдбаре.
4. **Тестирование (SDD):**
   - Расширен `test_collapsible_sidebar_desktop_feature` в [tests/unit/test_frontend_spa.py](file:///home/mariam/WebstormProjects/Errorium/tests/unit/test_frontend_spa.py).

### Результаты верификации:
- Unit-тесты `python3 -m unittest tests/unit/test_frontend_spa.py`: **PASS (11 unit tests PASS)**.

## [2026-10-08] — Возможность сворачивания бокового меню на десктопе по кнопке с фирменной SVG-иконкой

### Затронутые компоненты и спецификации:
- `docs/Product/04-interface-and-behavior.md`
- `tests/unit/test_frontend_spa.py`
- `services/frontend/static/index.html`
- `services/frontend/static/styles.css`
- `services/frontend/static/app.js`
- `STATUS.md`

### Суть изменений:
1. **Кнопка сворачивания/разворачивания бокового меню (`.sidebar-toggle-btn` и `.header-sidebar-toggle-btn`):**
   - Добавлена кнопка со специализированной SVG-иконкой панели (`M6.5 17.5L6.5 6.5...`) в шапку сайдбара и шапку приложения.
   - Стилизована обводка `stroke: currentColor !important`, обеспечивающая адаптацию под тему оформления и подсветку при наведении.
2. **Логика и стили сворачивания на широких экранах:**
   - По умолчанию боковое меню гарантированно открыто (раскрыто) на всех экранах от 768px и выше.
   - Состояние сохраняется в `localStorage` по ключу `errorium_sidebar_state` (`expanded` / `collapsed`).
   - Изначально (при первом визите или отсутствии явного сохранения) сайдбар всегда раскрыт. Состояние сворачивания восстанавливается только если пользователь явно свернул его ранее.
   - Очищен устаревший ключ `errorium_sidebar_collapsed`, чтобы исключить ошибочное открытие в свернутом виде из-за предыдущих тестовых кликов.
   - По клику на кнопку в сайдбаре меню плавно сворачивается влево (`margin-left: -var(--sidebar-width)`), расширяя рабочую область на 100% ширины экрана.
   - В шапке основного экрана при свернутом сайдбаре отображаются кнопка разворачивания (с аналогичной SVG-иконкой) и логотип `Errorium`.
   - Повторный клик разворачивает меню в исходное состояние. Добавлено сочетание клавиш `Ctrl+B` / `Cmd+B`.
3. **Тестирование (SDD):**
   - Добавлен тест `test_collapsible_sidebar_desktop_feature` в `tests/unit/test_frontend_spa.py`, проверяющий разметку, SVG, стили и логику управления состоянием в localStorage.

### Результаты верификации:
- Unit-тесты `python3 -m unittest discover tests/unit`: **PASS (11 unit tests PASS)**.

## [2026-10-08] — Адаптивное масштабирование синего огонька маскота на экранах с малой высотой (Samsung Galaxy Fold)

### Затронутые компоненты и спецификации:
- `docs/Product/04-interface-and-behavior.md`
- `tests/unit/test_frontend_spa.py`
- `services/frontend/static/styles.css`
- `STATUS.md`

### Суть изменений:
1. **Ограничение размеров `.hero-glow-aura` и `.blue-fire-bg`:**
   - По умолчанию максимальный размер ауры уменьшен с чрезмерных `900px / 140vw` до безопасных `min(520px, 120vw)` с ограничением `max-height: 56vh`.
   - Для фонового изображения `.blue-fire-bg` включено свойство `object-fit: contain; max-height: 100%; max-width: 100%;`, предотвращающее растяжение по вертикали.
2. **Медиа-запросы для экранов с малой высотой (Galaxy Fold cover screen / Landscape):**
   - Добавлен `@media (max-height: 720px)`: уменьшение ширины ауры до `380px` (`max-width: 90vw; max-height: 48vh;`).
   - Добавлен `@media (max-height: 600px)`: уменьшение ширины ауры до `290px` (`max-width: 80vw; max-height: 40vh;`), маскота до `84px` и компактные отступы секции.
3. **Тестирование (SDD):**
   - Добавлен модульный тест `test_short_viewport_height_scaling` в `tests/unit/test_frontend_spa.py`.

### Результаты верификации:
- Unit-тесты `make test-unit`: **PASS (Go PASS, Python SPA 9 tests PASS 0.56s)**.

## [2026-10-08] — Адаптация бокового меню для широких экранов: постоянный сайдбар и скрытие дублирующего логотипа

### Затронутые компоненты и спецификации:
- `docs/Product/04-interface-and-behavior.md`
- `tests/unit/test_frontend_spa.py`
- `services/frontend/static/index.html`
- `services/frontend/static/styles.css`
- `services/frontend/static/app.js`
- `STATUS.md`

### Суть изменений:
1. **Постоянная боковая панель недавних чатов на широких экранах (≥1024px):**
   - Сайдбар больше не сворачивается в мобильный Drawer, а отображается в статичном лейауте слева (`position: relative`, `transform: none`, `visibility: visible`, `width: var(--sidebar-width)`).
   - Затемняющий фон-бэкдроп (`.drawer-backdrop`) скрыт на широких экранах (`display: none`).
   - Кнопка закрытия `✕` (`.close-btn`) в шапке сайдбара скрыта, так как панель больше не закрывается.
2. **Исключение дублирования бренда Errorium в интерфейсе:**
   - Посередине экрана в шапке основного контента заголовок `Errorium` (`.header-logo`) скрыт, исключая повторение названия проекта (название остается единственным — в шапке боковой панели).
   - Кнопка гамбургер-меню (`.header-menu-btn`) скрыта на широких экранах.
   - Оставшаяся кнопка профиля/аутентификации аккуратно выровнена по правому краю шапки (`justify-content: flex-end`).
   - Логотип `Errorium` в шапке сайдбара сделан интерактивным и кликабельным (клик переключает на Hero-экран / создание нового чата).
3. **Безопасное управление состоянием в JavaScript:**
   - Добавлена проверка ширины окна в `openSidebar`, предотвращающая блокировку `overflow: hidden` на десктопе.
4. **Тестирование (SDD):**
   - Добавлен модульный тест `test_desktop_sidebar_and_header_rules` в `tests/unit/test_frontend_spa.py`.

### Результаты верификации:
- Unit-тесты `make test-unit`: **PASS (Go PASS, Python SPA 8 tests PASS 0.54s)**.

## [2026-10-08] — Реализация адаптивного Web Frontend SPA по дизайн-референсам (No-Frameworks)

### Затронутые компоненты и спецификации:
- `services/frontend/static/index.html`
- `services/frontend/static/styles.css`
- `services/frontend/static/app.js`
- `services/frontend/static/mascot.png`
- `services/frontend/static/blue-fire.png`
- `services/frontend/dev_server.py`
- `services/frontend/nginx.dev.conf`
- `docker-compose.dev.yml`
- `services/frontend/README.md`
- `docs/Tech/02-running.md`
- `docs/Product/04-interface-and-behavior.md`
- `tests/unit/test_frontend_spa.py`
- `Makefile`
- `STATUS.md`

### Суть изменений:
1. **Перенос дизайна точь-в-точь по макетам из `references/`:**
   - **Экран `Chat-screen.png`:**
     - Шапка с кнопкой меню (гамбургер), центрированным заголовком `Errorium` и иконкой профиля.
     - Центральный Hero-блок с парящим фирменным маскотом `mascot.png` на фоне оригинального арта `blue-fire.png` (интегрирующего синюю ауру и световой эллипс спотлайта), фирменными слоганами в моноширинном шрифте:
       `Element 404. Reacts with developers` (с красным акцентом на `404.`) и `Stack overflow. Issue overknown.`.
     - Капсульный нижний контейнер ввода с кнопкой `+`, автоувеличением высоты textarea, синей кнопкой отправки `↑` и предупреждением `AI can make mistakes. Check important information.`.
   - **Боковое меню `Chat-screen-menu-not-authorized.png`:**
     - Выдвижная панель (Drawer) с анимацией и блюр-бэкдропом.
     - Поле поиска по недавним чатам `Search`.
     - Секция `Recent chats` со стрелкой и 7 референсными ошибками (SyntaxError, IndentationError, TypeError, NameError, UnboundLocalError, KeyError) с меню `⋮`.
     - Навигационные пункты `MCP` и `About us`.
     - Плавающая белая пилюля `New chat` с иконкой пера/блокнота.
     - Нижняя строка профиля `Log in` / `Meet new features!` и иконка настроек `⚙`.
2. **Чистая архитектура SPA без сторонних фреймворков:**
   - Полное отсутствие React, Vue, Angular, Bootstrap, Tailwind или внешних CDN.
   - Модульный Vanilla JavaScript (`services/frontend/static/app.js`) с базой знаний типичных ошибок, генерацией ответов RAG, подсветкой синтаксиса и копированием блоков кода.
   - Модальные окна интеграции Model Context Protocol (MCP) с выдачей персонального Bearer-токена и готового сниппета `mcp.json` для Cursor / Claude Code.
3. **Адаптивность (Mobile-First):**
   - Пиксель-перфект верстка под мобильные устройства (390-430px) и плавное масштабирование на планшеты и десктопы.
4. **Тестирование (SDD):**
   - Создан тест `tests/unit/test_frontend_spa.py`, проверяющий наличие ассетов, отсутствие фреймворков, соответствие референсным текстам и адаптивность CSS.
   - Таргет `make test-unit` расширен для запуска тестов фронтенда.
5. **Устранение горизонтального скролла и окрашивание интерфейса браузера:**
   - Добавлены мета-теги `theme-color` (`#070a13`) для светлой и темной тем, `apple-mobile-web-app-status-bar-style: black-translucent` и `viewport-fit=cover` для бесшовной интеграции со статус-баром телефона.
   - Настроены ограничения `overflow-x: hidden` на `html, body, .app-container, .main-viewport, .view-content, .hero-screen`.
   - Закрытое боковое меню получило `visibility: hidden; pointer-events: none; box-shadow: none;` для предотвращения расширения виртуального скролла на мобильных устройствах.
   - Текст слогана и фон `blue-fire.png` адаптированы через `clamp()` и `max-width: 94vw`.

### Результаты верификации:
- Unit-тесты `make test-unit`: **PASS (Go tests PASS 1.014s, Python SPA tests PASS 0.001s)**.
- Docker build `services/frontend`: **Успешно (2.6s)**.

## [2026-10-01] — Настройка CI/CD автодеплоя (Gateway + Frontend)

### Затронутые компоненты и спецификации:
- `services/gateway/`
- `services/frontend/`
- `docker-compose.yml`
- `.github/workflows/deploy.yml`
- `Makefile`
- `STATUS.md`

### Суть изменений:
1. **Gateway Service (Go 1.27.1):**
   - Развернута модульная архитектура:
     - `pkg/config/config.go` — загрузка конфигурации окружения (`PORT`).
     - `domain/` — заглушка доменного слоя.
     - `internal/handler/health.go` — регистрация маршрутов и обработчик `GET /health` -> `{"status":"ok"}`.
     - `internal/app/app.go` — фабрика экземпляра приложения `App.New(cfg)` с регистрацией хэндлеров, запуск и `Shutdown(ctx)`.
     - `cmd/main.go` — точка входа с инициализацией конфига, приложения и graceful shutdown.
   - Написаны модульные тесты `internal/handler/health_test.go` (`GET /health` и метод `POST` -> 405).
   - Создан мультистейдж `Dockerfile` (`golang:1.27.1-alpine` -> `alpine:3.20`).

2. **Frontend Service (Nginx 1.26-alpine):**
   - Создан production-ready `nginx.conf`:
     - Редирект с HTTP (порт 80) на HTTPS (порт 443) для домена `errorium.ru`.
     - Настройка SSL (Mozilla Intermediate, ciphers, TLSv1.2/1.3, сессионный кэш).
     - Gzip-сжатие (уровень 6, типы json, js, css, xml, svg).
     - Проксирование API-запросов (`/api/` и `/health`) на шлюз `gateway:8000`.
   - Создана минималистичная заглушка `static/index.html` («Errorium. Скоро»).
   - Создан `Dockerfile` на базе `nginx:1.26-alpine`.

3. **Инфраструктура Docker Compose (`docker-compose.yml`):**
   - Сконфигурированы сервисы `gateway` и `frontend`.
   - Использованы теги образов GHCR (`ghcr.io/soskiiit/errorium-<service>:develop`) и локальные контексты сборки.
   - Настроено монтирование SSL-сертификатов хоста (`/etc/ssl/errorium.crt`, `/etc/ssl/errorium.key`) в контейнер фронтенда.
   - Поисковое ядро (`search_engine_core` и `search_engine_indexer`) временно исключено по согласованию.

4. **GitHub Actions CI/CD (`.github/workflows/deploy.yml`):**
   - `Stage 0 (test)`: запуск модульных тестов Go (`go test -v -race ./...`).
   - `Stage 1 (build-and-push)`: сборка и пуш Docker-образов в GHCR по триггеру `push` в `develop` или `workflow_dispatch` (запускается строго после тестов).
   - `Stage 2 (deploy)`: доставка `docker-compose.yml` на dev-сервер по SCP и перезапуск сервисов по SSH (`docker compose pull && docker compose up -d --remove-orphans`).

5. **Makefile:**
   - Обновлен таргет `test-unit` для запуска тестов `gateway` на Go.

### Результаты верификации:
- Unit-тесты `go test -v -race ./...`: **PASS (1.01s)**.
- Сборка Dockerfile `gateway`: **Успешно**.
- Сборка Dockerfile `frontend`: **Успешно**.

## [2026-10-01] — Настройка SEO, robots.txt, sitemap.xml и брендинга

### Затронутые компоненты:
- `services/frontend/static/robots.txt`
- `services/frontend/static/sitemap.xml`
- `services/frontend/static/favicon.svg`
- `services/frontend/static/index.html`
- `.gitignore`

### Суть изменений:
1. Создан `robots.txt` с разрешением индексации контента и запретом служебных путей (`/api/`, `/health`, `/admin/`).
2. Создан `sitemap.xml` для домена `https://errorium.ru/` с ежедневным приоритетом.
3. Добавлено исключение в `.gitignore` для `services/frontend/static/sitemap.xml`.
4. Создан брендовый векторный фавикон `favicon.svg`.
5. Обновлен `index.html` со слоганом «Errorium. Your error has been solved before.», полным набором мета-тегов для поисковиков, Open Graph и Twitter Cards.
