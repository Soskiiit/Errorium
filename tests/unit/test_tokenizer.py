"""Unit-тесты токенизатора индексатора (docs/Tech/03-data.md, раздел 5).

Эталонные примеры из п. 5.5 спецификации; запрос и документ обязаны проходить
один и тот же токенизатор (методичка, замечание 1.1).
"""
import os
import sys
import unittest

REPO_ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "../.."))
sys.path.insert(0, os.path.join(REPO_ROOT, "services", "search_engine_indexer", "src"))

from indexer.tokenizer import TOKENIZER_VERSION, clean_html, tokenize  # noqa: E402


class TestReferenceExamples(unittest.TestCase):
    """Таблица эталонных примеров (п. 5.5): вход -> токены в порядке эмиссии."""

    CASES = [
        ("NullPointerException",
         ["nullpointerexception", "null", "pointer", "exception"]),
        ("IndexOutOfBoundsException",
         ["indexoutofboundsexception", "index", "out", "of", "bounds", "exception"]),
        ("ZeroDivisionError",
         ["zerodivisionerror", "zero", "division", "error"]),
        ("ECONNREFUSED", ["econnrefused"]),
        ("HTTPServer", ["httpserver", "http", "server"]),
        ("C++ vs C#", ["c++", "vs", "c#"]),
        ("in .NET", ["in", ".net", "net"]),
        ("pip3 install docker-compose",
         ["pip3", "install", "docker-compose", "docker", "compose"]),
        ("g++ -o app main.cpp", ["g++", "app", "main.cpp", "main", "cpp"]),
        ('fmt.Println("hi")', ["fmt.println", "fmt", "println", "hi"]),
        ("os.path.join(a, b)", ["os.path.join", "os", "path", "join"]),
        ("at com.example.Foo.bar(Foo.java:42)",
         ["at", "com.example.foo.bar", "com", "example", "foo", "bar",
          "foo.java", "foo", "java", "42"]),
        ("HTTP 404, see https://x.io/a", ["http", "404", "see"]),
        ("my_var = 1", ["my_var", "my", "var"]),
        ("a.b", ["a.b"]),
    ]

    def test_reference_table(self):
        for text, expected in self.CASES:
            with self.subTest(text=text):
                self.assertEqual(tokenize(text), expected)


class TestIndivisibleTokens(unittest.TestCase):
    """AGENTS.md п. 2.3: идентификаторы, символы языков и утилиты неделимы."""

    def test_error_identifiers_kept_whole(self):
        for ident in ("NullPointerException", "IndexOutOfBoundsException", "ZeroDivisionError"):
            with self.subTest(ident=ident):
                self.assertIn(ident.lower(), tokenize(f"Caught {ident} in thread main"))

    def test_language_symbols_kept_whole(self):
        tokens = tokenize("Compare C++ and C# on .NET with fmt.Println and os.path.join")
        for expected in ("c++", "c#", ".net", "fmt.println", "os.path.join"):
            self.assertIn(expected, tokens)

    def test_utilities_kept_whole(self):
        tokens = tokenize("run pip3 and g++ then docker-compose up")
        for expected in ("pip3", "g++", "docker-compose"):
            self.assertIn(expected, tokens)

    def test_whole_token_precedes_its_parts(self):
        tokens = tokenize("NullPointerException")
        self.assertEqual(tokens[0], "nullpointerexception")

    def test_no_split_on_digits_or_all_caps(self):
        self.assertEqual(tokenize("pip3"), ["pip3"])
        self.assertEqual(tokenize("utf8"), ["utf8"])
        self.assertEqual(tokenize("0x80070005"), ["0x80070005"])
        self.assertEqual(tokenize("ECONNREFUSED"), ["econnrefused"])


class TestNoStemming(unittest.TestCase):
    """П. 5.3: без стемминга, лемматизации и стоп-слов."""

    def test_word_forms_not_merged(self):
        self.assertEqual(tokenize("running exceptions errors"), ["running", "exceptions", "errors"])

    def test_stopwords_kept(self):
        self.assertEqual(tokenize("how to fix the error"), ["how", "to", "fix", "the", "error"])


class TestPunctuationAndEdges(unittest.TestCase):
    def test_trailing_dot_and_hyphen_stripped(self):
        self.assertEqual(tokenize("error."), ["error"])
        self.assertEqual(tokenize("docker-compose."), ["docker-compose", "docker", "compose"])
        self.assertEqual(tokenize("C++."), ["c++"])

    def test_punctuation_is_separator(self):
        self.assertEqual(tokenize("foo(bar);baz[qux]"), ["foo", "bar", "baz", "qux"])

    def test_numbers_kept_single_chars_dropped(self):
        self.assertEqual(tokenize("error 404 and 500 at 7"), ["error", "404", "and", "500", "at"])

    def test_long_tokens_dropped(self):
        self.assertEqual(tokenize("a" * 65 + " ok"), ["ok"])
        self.assertEqual(tokenize("b" * 64), ["b" * 64])

    def test_url_removed_text_kept(self):
        self.assertEqual(tokenize("see https://example.com/a?b=1 now"), ["see", "now"])

    def test_empty_and_whitespace(self):
        self.assertEqual(tokenize(""), [])
        self.assertEqual(tokenize("   \n\t "), [])

    def test_duplicates_preserved_for_tf(self):
        self.assertEqual(tokenize("graph graph Graph"), ["graph", "graph", "graph"])

    def test_unicode_lowercased(self):
        self.assertEqual(tokenize("Ошибка компиляции"), ["ошибка", "компиляции"])

    def test_deterministic(self):
        text = "NullPointerException at com.example.Foo.bar(Foo.java:42)"
        self.assertEqual(tokenize(text), tokenize(text))

    def test_query_with_angle_brackets_not_treated_as_html(self):
        self.assertIn("list", tokenize("List<String> cannot be cast"))
        self.assertIn("string", tokenize("List<String> cannot be cast"))


class TestCleanHtml(unittest.TestCase):
    def test_tags_removed_code_content_kept(self):
        html = "<p>Use <code>os.path.join</code> here</p><pre><code>x = os.path.join(a, b)</code></pre>"
        text = clean_html(html)
        self.assertNotIn("<p>", text)
        self.assertNotIn("<code>", text)
        self.assertEqual(tokenize(text).count("os.path.join"), 2)

    def test_entities_unescaped_after_tag_removal(self):
        text = clean_html("<p>Use <code>List&lt;String&gt;</code> &amp; more</p>")
        self.assertIn("List<String>", text)
        self.assertIn("&", text)
        self.assertNotIn("&lt;", text)
        # скобки из сущностей не должны быть приняты за теги
        tokens = tokenize(text)
        self.assertIn("list", tokens)
        self.assertIn("string", tokens)

    def test_paragraphs_do_not_glue_words(self):
        tokens = tokenize(clean_html("<p>first</p><p>second</p>"))
        self.assertEqual(tokens, ["first", "second"])

    def test_link_text_kept_url_removed(self):
        html = '<a href="https://example.com/x">the docs</a>'
        self.assertEqual(tokenize(clean_html(html)), ["the", "docs"])


class TestVersion(unittest.TestCase):
    def test_version_constant(self):
        self.assertEqual(TOKENIZER_VERSION, "1")


if __name__ == "__main__":
    unittest.main()
