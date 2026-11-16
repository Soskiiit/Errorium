"""Unit-тесты экрана Star Map (FR-3.4, FR-3.5, ADR-006).

Проверяют структуру разметки, стили и логику интерактивной карты связей
SO-тредов: кнопка входа в карточке RAG-ответа, анимированный уход чата и
prompt bar, Canvas-визуализация с собственной force-симуляцией, мок-данные
совместимые с контрактами protos/graph.proto.
"""

import os
import re
import shutil
import subprocess
import unittest

REPO_ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "../.."))
STATIC_DIR = os.path.join(REPO_ROOT, "services", "frontend", "static")


def read_static(name):
    with open(os.path.join(STATIC_DIR, name), "r", encoding="utf-8") as f:
        return f.read()


class TestStarMap(unittest.TestCase):
    def test_star_map_view_structure(self):
        """Разметка экрана Star Map: canvas, шапка, легенда, zoom, тултип, панель деталей."""
        html = read_static("index.html")

        self.assertIn('id="star-map-view"', html)
        self.assertIn('id="star-map-canvas"', html)
        self.assertIn('id="btn-star-map-back"', html)
        self.assertIn("Back to chat", html)
        self.assertIn("Star Map", html)
        self.assertIn('id="star-map-subtitle"', html)
        self.assertIn('id="star-map-legend"', html)
        self.assertIn('id="btn-star-map-zoom-in"', html)
        self.assertIn('id="btn-star-map-zoom-out"', html)
        self.assertIn('id="btn-star-map-fit"', html)
        self.assertIn('id="star-map-tooltip"', html)
        self.assertIn('id="star-map-detail-panel"', html)
        self.assertIn('id="btn-star-map-detail-close"', html)
        self.assertIn('id="star-map-detail-title"', html)
        self.assertIn('id="star-map-detail-answer"', html)
        self.assertIn('id="star-map-detail-link"', html)
        self.assertIn("Open on Stack Overflow", html)
        self.assertIn("Drag nodes", html)

        # Легенда: четыре категории FR-3.2
        for label in ["Top result", "Top-K results", "Duplicates", "Linked in answers"]:
            self.assertIn(label, html, f"Категория легенды не найдена: {label}")

    def test_star_map_entry_button_in_assistant_card(self):
        """Кнопка Star Map рендерится в шапке карточки RAG-ответа (FR-3.4)."""
        js = read_static("app.js")

        self.assertIn("star-map-open-btn", js)
        self.assertIn("openStarMapFromCard", js)
        self.assertIn("window.openStarMapFromCard", js)

        # Кнопка располагается внутри card-top-bar карточки ассистента
        top_bar_match = re.search(
            r'<div class="card-top-bar">.*?</div>\s*</div>', js, re.DOTALL
        )
        self.assertIsNotNone(top_bar_match, "card-top-bar должен присутствовать в шаблоне renderConversation")
        self.assertIn("star-map-open-btn", top_bar_match.group(0))

    def test_star_map_transition_animations(self):
        """Анимированный уход чата и prompt bar и появление карты (FR-3.4)."""
        js = read_static("app.js")
        css = read_static("styles.css")

        # Классы анимаций в JS
        self.assertIn("map-leaving", js)
        self.assertIn("map-entering", js)
        self.assertIn("star-map-active", js)

        # Keyframes и классы в CSS
        for keyframe in ["starMapChatLeave", "starMapFooterLeave", "starMapEnter", "starMapLeave"]:
            self.assertIn(f"@keyframes {keyframe}", css, f"Нет keyframes {keyframe}")
        self.assertIn(".chat-screen.map-leaving", css)
        self.assertIn(".app-prompt-footer.map-leaving", css)
        self.assertIn(".star-map-screen.map-entering", css)
        self.assertIn(".star-map-screen.map-leaving", css)

        # Анимации содержат fade + scale + blur
        enter_match = re.search(r"@keyframes starMapEnter\s*\{(.*?)\}\s*\}", css, re.DOTALL)
        self.assertIsNotNone(enter_match)
        enter_css = enter_match.group(1)
        self.assertIn("opacity", enter_css)
        self.assertIn("scale", enter_css)
        self.assertIn("blur", enter_css)

        # На время карты маска нижнего затемнения view-content снимается
        self.assertIn("body.star-map-active .view-content", css)

    def test_star_map_mock_data_contract(self):
        """Мок-данные совместимы с ThreadGraphResponse / ThreadNode (FR-3.5, graph.proto)."""
        js = read_static("app.js")

        self.assertIn("MOCK_RELATED_THREADS", js)
        self.assertIn("buildStarMapGraph", js)

        # Категории узлов из NodeCategory (graph.proto)
        for category in ["PRIMARY_RESULT", "OTHER_RESULT", "DUPLICATE", "LINKED"]:
            self.assertIn(category, js, f"Категория узла не найдена: {category}")

        # Типы ребер из EdgeType (graph.proto)
        for edge_type in ["DUPLICATE_OF", "LINKED_IN_ANSWER"]:
            self.assertIn(edge_type, js, f"Тип ребра не найден: {edge_type}")

        # Поля ThreadNode
        for field in ["doc_id", "score", "title", "snippet", "has_accepted_answer", "tags", "best_answer_score"]:
            self.assertIn(field, js, f"Поле ThreadNode не найдено: {field}")

        # Поля ThreadDetailsResponse (панель деталей по клику)
        self.assertIn("best_answer_body", js)
        self.assertIn("original_url", js)

    def test_star_map_canvas_engine_no_external_libs(self):
        """Canvas-движок с собственной force-симуляцией, без D3/Cytoscape/React Flow (ADR-006)."""
        js = read_static("app.js")

        self.assertIn("requestAnimationFrame", js)
        self.assertIn("getContext('2d')", js)

        # Собственная физика: отталкивание, пружины, затухание
        self.assertIn("repulsion", js)
        self.assertIn("spring", js)
        self.assertIn("damping", js)
        self.assertIn("alpha", js)

        # Интерактив: pointer events, zoom, drag
        self.assertIn("pointerdown", js)
        self.assertIn("pointermove", js)
        self.assertIn("pointerup", js)
        self.assertIn("wheel", js)

        # Запрет готовых библиотек визуализации
        for lib in ["d3.js", "d3.min", "cytoscape", "react-flow", "reactflow", "sigma.js", "vis-network"]:
            self.assertNotIn(lib, js.lower(), f"Обнаружена запрещенная библиотека визуализации: {lib}")

    def test_star_map_node_colors_and_size(self):
        """Цветовое кодирование FR-3.2 и размер узла пропорциональный sqrt(score)."""
        css = read_static("styles.css")
        js = read_static("app.js")

        for color in ["#10b981", "#f59e0b", "#38bdf8", "#a855f7"]:
            self.assertIn(color, css, f"Цвет категории FR-3.2 не найден в CSS: {color}")
            self.assertIn(color, js, f"Цвет категории FR-3.2 не найден в JS: {color}")

        self.assertIn("Math.sqrt", js)
        self.assertIn("score", js)

    def test_star_map_hover_and_click_behavior(self):
        """Hover-тултип (FR-3.3) и клик с панелью деталей принятого ответа, без рейтинга и галочки."""
        js = read_static("app.js")
        css = read_static("styles.css")
        html = read_static("index.html")

        self.assertIn("starMapTooltip", js)
        self.assertIn("hoverNode", js)
        self.assertIn("selectedNode", js)
        self.assertIn("has_accepted_answer", js)

        # Рейтинг и галочка убраны со всех поверхностей карты
        # (секция Star Map; ✅ в мок-коде чата KNOWLEDGE_BASE не относится к карте)
        star_map_section = js[js.index("const STAR_MAP_COLORS"):js.index("// Animated view transitions")]
        self.assertNotIn("✅", star_map_section)
        self.assertNotIn("▲", star_map_section)
        self.assertNotIn("star-map-tooltip-meta", js)
        self.assertNotIn("star-map-detail-meta", js)
        self.assertNotIn("star-map-detail-meta", html)

        self.assertIn(".star-map-tooltip", css)
        self.assertIn(".star-map-detail-panel", css)
        self.assertIn(".star-map-detail-panel.open", css)

    def test_star_map_routing_and_escape(self):
        """Маршрут /star-map через History API и закрытие по Esc (FR-3.4)."""
        js = read_static("app.js")

        self.assertIn("'/star-map'", js)
        self.assertIn("openStarMap", js)
        self.assertIn("closeStarMap", js)
        self.assertIn("hideStarMapImmediate", js)

        # Esc закрывает карту
        esc_match = re.search(r"if \(e\.key === 'Escape'\) \{(.*?)\}", js, re.DOTALL)
        self.assertIsNotNone(esc_match, "Обработчик Escape должен существовать")
        self.assertIn("closeStarMap", esc_match.group(1))

    def test_star_map_styles_design_system(self):
        """Стили карты в дизайн-системе Errorium: стекло, аура, адаптивность."""
        css = read_static("styles.css")

        self.assertIn(".star-map-screen", css)
        self.assertIn(".star-map-canvas", css)
        self.assertIn(".star-map-aura", css)
        self.assertIn(".star-map-open-btn", css)
        self.assertIn(".star-map-legend", css)
        self.assertIn(".star-map-back-btn", css)
        self.assertIn("backdrop-filter", css)

        # Адаптивность панели деталей на мобильных
        self.assertIn("@media (max-width: 767px)", css)

        # Уважение prefers-reduced-motion
        self.assertIn("prefers-reduced-motion", css)

    def test_star_map_star_shaped_nodes(self):
        """Узлы — маленькие четырёхлучевые звёзды со свечением своего цвета, а не разноцветные кружки."""
        js = read_static("app.js")
        css = read_static("styles.css")

        # Sparkle-путь узла (quadratic curves) и цветной ореол
        self.assertIn("starPath", js)
        self.assertIn("quadraticCurveTo", js)
        self.assertIn("createRadialGradient", js)
        self.assertIn("colorWithAlpha", js)

        # Мерцание и лёгкий детерминированный наклон звёзд
        self.assertIn("twinklePhase", js)
        self.assertIn("twinkleSpeed", js)
        self.assertIn("tilt", js)

        # Звёздчатые маркеры легенды (clip-path + drop-shadow вместо круглых точек)
        dot_match = re.search(r"\.star-map-dot\s*\{([^}]+)\}", css)
        self.assertIsNotNone(dot_match, "Селектор .star-map-dot должен присутствовать")
        self.assertIn("clip-path: polygon", dot_match.group(1))
        self.assertIn("drop-shadow", css)

    def test_star_map_progressive_labels_and_compact_stars(self):
        """Компактные звёзды с белой мини-звездой внутри и прогрессивные подписи по зуму."""
        js = read_static("app.js")

        # Уменьшенный радиус узлов
        self.assertIn("4 + Math.sqrt(Math.max(1, n.score)) * 0.15", js)
        self.assertIn("Math.min(17, Math.max(5,", js)

        # Белая мини-звезда внутри вместо белого круга, соосно телу (без поворота)
        self.assertNotIn("ctx.rotate(Math.PI / 4)", js)
        self.assertIn("const coreR = Math.max(2, r * 0.32)", js)
        self.assertIn("starPath(ctx, coreR)", js)

        # Мягкие размытые края ядра: radial gradient к кончикам лучей + белое свечение
        self.assertIn("coreGrad", js)
        self.assertIn("rgba(255, 255, 255, 0)", js)

        # Ступени подписей: сокращённые -> содержательные
        self.assertIn("STAR_MAP_LABEL_TIERS", js)
        self.assertIn("{ k: 0.7, len: 12 }", js)
        self.assertIn("{ k: 1.15, len: 26 }", js)
        self.assertIn("{ k: 1.8, len: 48 }", js)

        # Crossfade между ступенями; рейтинг и галочка в подписях отсутствуют
        self.assertIn("1 - ramp", js)
        self.assertNotIn("▲ ${n.score}", js)

    def test_star_map_smooth_zoom_buttons(self):
        """Плавный анимированный зум по кнопкам +/−/⤢ с накоплением и прерыванием ручным вводом."""
        js = read_static("app.js")

        # Анимация вида и easing
        self.assertIn("animateViewTo", js)
        self.assertIn("applyViewAnimation", js)
        self.assertIn("easeOutCubic", js)
        self.assertIn("zoomAnim", js)

        # Плавные варианты wired на кнопки
        self.assertIn("StarMap.zoomSmooth(StarMap.w / 2, StarMap.h / 2, 1.25)", js)
        self.assertIn("StarMap.zoomSmooth(StarMap.w / 2, StarMap.h / 2, 0.8)", js)
        self.assertIn("StarMap.fitViewSmooth()", js)

        # Прерывание анимации ручным взаимодействием (pointerdown и wheel)
        down_match = re.search(r"addEventListener\('pointerdown', \(e\) => \{(.*?)\n            \}\);", js, re.DOTALL)
        self.assertIsNotNone(down_match, "Обработчик pointerdown должен существовать")
        self.assertIn("this.zoomAnim = null", down_match.group(1))

        wheel_match = re.search(r"addEventListener\('wheel', \(e\) => \{(.*?)\}, \{ passive: false \}\);", js, re.DOTALL)
        self.assertIsNotNone(wheel_match, "Обработчик wheel должен существовать")
        self.assertIn("this.zoomAnim = null", wheel_match.group(1))

        # prefers-reduced-motion: мгновенное применение без анимации
        anim_match = re.search(r"animateViewTo\(target, duration = 280\) \{(.*?)\n        \},", js, re.DOTALL)
        self.assertIsNotNone(anim_match, "animateViewTo должен существовать")
        self.assertIn("reducedMotion", anim_match.group(1))

    def test_star_map_i18n(self):
        """Локализация EN/RU ключевых строк карты."""
        js = read_static("app.js")

        self.assertIn('"Star Map": "Карта связей"', js)
        self.assertIn('"Карта связей": "Star Map"', js)
        self.assertIn('"Back to chat": "Назад в чат"', js)
        self.assertIn('"Open on Stack Overflow": "Открыть на Stack Overflow"', js)

    def test_app_js_syntax_with_star_map(self):
        """Синтаксическая валидность app.js с движком Star Map через node --check."""
        js_path = os.path.join(STATIC_DIR, "app.js")
        node_bin = shutil.which("node")
        if not node_bin:
            self.skipTest("Node.js не установлен в текущем окружении для запуска node --check")
        res = subprocess.run([node_bin, "--check", js_path], capture_output=True, text=True)
        self.assertEqual(
            res.returncode, 0,
            f"Синтаксическая ошибка в {js_path}:\nSTDOUT:\n{res.stdout}\nSTDERR:\n{res.stderr}",
        )


if __name__ == "__main__":
    unittest.main()