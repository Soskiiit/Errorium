from pathlib import Path
import unittest


ROOT = Path(__file__).resolve().parents[2]
STATIC = ROOT / "services" / "frontend" / "static"


class FrontendLayoutContractTest(unittest.TestCase):
    def test_chat_shell_and_guest_menu_exist(self):
        html = (STATIC / "index.html").read_text(encoding="utf-8")
        self.assertIn('class="sidebar"', html)
        self.assertIn('class="chat-main"', html)
        self.assertIn("Recent chats", html)
        self.assertIn("Log in", html)
        self.assertIn('aria-label="Open menu"', html)
        self.assertIn('rel="canonical"', html)
        self.assertIn('property="og:title"', html)
        self.assertIn('src="/mascot.png"', html)
        self.assertIn('src="/blue-fire.png"', html)
        self.assertIn('class="hero-bg"', html)
        self.assertIn('class="composer"', html)
        self.assertNotIn("chatgpt.com", html)
        self.assertTrue((STATIC / "mascot.png").is_file())
        self.assertTrue((STATIC / "blue-fire.png").is_file())

    def test_responsive_breakpoint_and_desktop_sidebar(self):
        css = (STATIC / "styles.css").read_text(encoding="utf-8")
        self.assertIn("@media (min-width: 800px)", css)
        self.assertIn("--sidebar-width: 496px", css)
        self.assertIn(".hero-bg", css)
        self.assertIn(".mascot", css)
        self.assertNotIn(".spotlight", css)
        self.assertNotIn(".chat-main::before", css)

    def test_menu_interaction_is_wired(self):
        script = (STATIC / "app.js").read_text(encoding="utf-8")
        self.assertIn("Escape", script)
        self.assertIn("sidebar-open", script)
        self.assertIn("matchMedia", script)


if __name__ == "__main__":
    unittest.main()
