# CHANGE_HISTORY.md — История изменений проекта Errorium

## [2026-10-09] — Исправление плавности transition и устранение рывков на кнопках авторизации

### Затронутые компоненты и спецификации:
- `services/frontend/static/styles.css`
- `tests/unit/test_frontend_spa.py`
- `STATUS.md`

### Суть изменений:
1. **Причина резкости и неработающего перехода:**
   - В спецификации CSS градиенты (`linear-gradient`) не интерполируются напрямую через `transition: background`, из-за чего браузер мгновенно переключал слои.
   - Число и типы теней в `box-shadow` в исходном и hover-состоянии не совпадали (смешивание outer и inset теней), из-за чего браузер не мог построить промежуточные кадры анимации и сбрасывал тень скачком.
   - В `.auth-yandex-btn` стояло завышенное значение `box-shadow 1.3s ease`, создававшее рассинхрон с остальными свойствами.
2. **Реализация плавного 60 FPS перехода ([styles.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/styles.css)):**
   - Для эффекта беления добавлен псевдоэлемент `::before` с плавным переходом прозрачности `opacity: 0 -> 1` (`0.3s cubic-bezier(0.16, 1, 0.3, 1)`), что обеспечивает аппаратное сглаживание без скачков.
   - Стек `box-shadow` синхронизирован по слоям 1-в-1 между базовым состоянием и hover (базовое состояние содержит нулевой красный ореол `0 0 0px 0px rgba(252, 63, 29, 0)`, плавно раскрывающийся до `25px`), что восстановило идеальную математическую интерполяцию.
   - Цвет свечения зафиксирован на чистом красном Яндекс ID (`rgba(252, 63, 29, 0.35)`).
3. **Тестирование ([test_frontend_spa.py](file:///home/mariam/WebstormProjects/Errorium/tests/unit/test_frontend_spa.py)):**
   - Все 15 тестов пройдены успешно (`make test-unit`).

### Затронутые компоненты и спецификации:
- `docs/Product/04-interface-and-behavior.md`
- `services/frontend/static/styles.css`
- `tests/unit/test_frontend_spa.py`
- `STATUS.md`

### Суть изменений:
1. **Спецификация (SDD):**
   - В [04-interface-and-behavior.md](file:///home/mariam/WebstormProjects/Errorium/docs/Product/04-interface-and-behavior.md) зафиксировано поведение `:hover` кнопки `Log in`: эффект осветления стеклянного слоя и излучение красного неонового ореола.
2. **Стилизация ([styles.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/styles.css)):**
   - Для `.auth-login-btn` задана базовая граница `1px solid rgba(255, 255, 255, 0.15)` для плавного перехода без сдвига верстки.
   - Для `.auth-login-btn:hover` настроены стили по аналогии с `.auth-yandex-btn:hover`:
     - Светлая граница `border-color: rgba(255, 255, 255, 0.35)`.
     - Верхний стеклянный блик `inset 0 1.5px 1.5px 0 rgba(255, 255, 255, 0.45)` и светлое осветление градиента («белеет»).
     - Яркий красный неоновый ореол `0 0 25px 2px rgba(252, 63, 29, 0.25)` («светится красным»).
     - Нижний спотлайт `.auth-btn-spotlight` при наведении на кнопку также плавно переключается на теплый красный оттенок `rgba(252, 63, 29, 0.45)`.
3. **Тестирование ([test_frontend_spa.py](file:///home/mariam/WebstormProjects/Errorium/tests/unit/test_frontend_spa.py)):**
   - Добавлены проверки hover-стилей и красного свечения кнопки входа. Все 15 тестов пройдены (`make test-unit`).

## [2026-10-09] — Оптимизация и уменьшение окна авторизации для мобильных телефонов

### Затронутые компоненты и спецификации:
- `docs/Product/04-interface-and-behavior.md`
- `services/frontend/static/styles.css`
- `tests/unit/test_frontend_spa.py`
- `STATUS.md`

### Суть изменений:
1. **Спецификация (SDD):**
   - В [04-interface-and-behavior.md](file:///home/mariam/WebstormProjects/Errorium/docs/Product/04-interface-and-behavior.md) зафиксированы компактные размеры и адаптивные пропорции окна авторизации для экранов $\le 480\text{px}$ и экранов с небольшой высотой.
2. **Стилизация ([styles.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/styles.css)):**
   - Добавлен медиа-запрос `@media (max-width: 480px)`:
     - Максимальная ширина карточки снижена с `380px` до `326px`, скругление углов уменьшено до `26px`.
     - Внутренние отступы сжаты с `38px 28px 32px 28px` до `24px 18px 20px 18px`.
     - Маскот уменьшен до `58px`, заголовок `Errorium` уменьшен до `1.35rem`.
     - Высота полей ввода, кнопки `Log in` и кнопки `Continue with Yandex ID` уменьшена с `50px`/`48px` до `44px`.
     - Размеры бейджа Яндекс ID скорректированы до `26px`.
     - Отступы разделителя `or` и подвала сжаты для устранения лишней высоты.
   - Добавлено правило для мобильных устройств в ландшафтном режиме / с малой высотой вьюпорта (`@media (max-height: 680px) and (max-width: 480px)`), гарантирующее комфортный обзор без перекрытия экрана.
3. **Тестирование ([test_frontend_spa.py](file:///home/mariam/WebstormProjects/Errorium/tests/unit/test_frontend_spa.py)):**
   - Добавлен тест `test_auth_modal_mobile_scaling`. Все 15 тестов пройдены успешно (`make test-unit`).

## [2026-10-09] — Реализация всплывающего окна авторизации точь-в-точь по макету Auth-screen.png (Liquid Glass & Яндекс ID)

### Затронутые компоненты и спецификации:
- `docs/Product/04-interface-and-behavior.md`
- `services/frontend/static/index.html`
- `services/frontend/static/styles.css`
- `services/frontend/static/app.js`
- `tests/unit/test_frontend_spa.py`
- `STATUS.md`

### Суть изменений:
1. **Спецификация (SDD):**
   - В [04-interface-and-behavior.md](file:///home/mariam/WebstormProjects/Errorium/docs/Product/04-interface-and-behavior.md) обновлен раздел «Экран 4: Модальное окно аутентификации», регламентирующий точную геометрию, визуальные параметры, эффект Liquid Glass и аутентичный логотип Яндекс ID.
2. **Разметка и аутентичный логотип Яндекс ([index.html](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/index.html)):**
   - Интегрирована карточка `.auth-modal-card` с фирменным маскотом Errorium (`/mascot.png`) и жирным заголовком `Errorium`.
   - Добавлены поля ввода `Email` и `Password` с контурными иконками (пользователь и замок).
   - Реализован ряд опций с кастомным зеленым чекбоксом `Remember me` с белой галочкой и ссылкой `Forgot password?`.
   - Добавлена полноразмерная скругленная зеленая кнопка `Log in` (pill, 24px) со световым пятном-спотлайтом под ней.
   - Разделитель `or` с тонкими градиентными линиями.
   - Кнопка SSO-авторизации `Continue with Yandex ID` с аутентичным векторным логотипом Яндекса (`viewBox="0 0 38 38"`, `fill="#FC3F1D" fill-opacity="0.65"`, точные векторные пути литеры `Y`).
   - Нижний блок `New here? Sign up` с активной ссылкой регистрации.
3. **Стилизация Liquid Glass ([styles.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/styles.css)):**
   - Для модальной карточки `.auth-modal-card`, полей ввода `.auth-input-group` и кнопки `.auth-yandex-btn` полностью воспроизведена оптическая схема **Liquid Glass** из строки ввода сообщений:
     - Оптическая полупрозрачность многослойного стеклянного градиента (`rgba(255, 255, 255, 0.03)` $\rightarrow$ `rgba(15, 23, 42, 0.35)` $\rightarrow$ `rgba(10, 16, 32, 0.5)`).
     - Глубокое размытие и цветовая сатурация подложки: `backdrop-filter: blur(28px) saturate(210%) brightness(108%)`.
     - Тонкая преломляющая рамка: `1px solid rgba(255, 255, 255, 0.22)`.
     - Фаски преломления и внутренние блики: `inset 0 1.5px 1.5px rgba(255, 255, 255, 0.35)`, `inset 0 12px 28px -6px rgba(255, 255, 255, 0.15)`, `inset 0 -2px 6px rgba(147, 197, 253, 0.15)`, `inset 0 -16px 36px rgba(5, 10, 25, 0.55)`.
   - Зеленый чекбокс с белой галочкой, микроанимации фокуса и рассеянный спотлайт под кнопкой входа.
4. **Интерактивность ([app.js](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/app.js)):**
   - Подключена валидация формы, отправка по Enter и кнопке `Log in`, обработчик `Continue with Yandex ID`, вывод уведомлений toast и обновление профиля пользователя в боковом меню.
5. **Тестирование ([test_frontend_spa.py](file:///home/mariam/WebstormProjects/Errorium/tests/unit/test_frontend_spa.py)):**
   - Добавлены модульные тесты проверки наличия полей, точного SVG Яндекс ID (`fill-opacity="0.65"`), стилей Liquid Glass (`backdrop-filter`) и обработчиков.
   - Все 14 тестов успешно проходят (`make test-unit`).

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
