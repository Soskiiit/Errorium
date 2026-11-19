package search

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode"
)

// Fixture represents a Stack Overflow error thread in the in-memory pool.
type Fixture struct {
	DocID         int64
	Title         string
	URL           string
	Snippet       string
	Tags          []string
	QuestionScore int
	IsAccepted    bool
	AnswerBody    string
	CodeSnippet   string
}

// DefaultFixtures contains realistic Stack Overflow threads for common errors across languages.
var DefaultFixtures = []Fixture{
	{
		DocID:         27539308,
		Title:         "SyntaxError: invalid syntax in Python (Top causes & fixes)",
		URL:           "https://stackoverflow.com/questions/27539308",
		Snippet:       "This error occurs when Python parser encounters code that violates language grammar rules. The most common causes are missing colons after if, for, def statements or unclosed parentheses.",
		Tags:          []string{"python", "syntax-error", "python-3.x"},
		QuestionScore: 4180,
		IsAccepted:    true,
		AnswerBody:    "SyntaxError: invalid syntax happens during Python tokenization and parsing. Ensure that all compound statements (def, if, for, while, class) end with a colon (:), check that string quotes and brackets are properly closed, and avoid using reserved keywords as variable names.",
		CodeSnippet:   "def calculate_total(items):\n    total = 0\n    return total",
	},
	{
		DocID:         15478423,
		Title:         "Why am I getting SyntaxError on the line after my print statement?",
		URL:           "https://stackoverflow.com/questions/15478423",
		Snippet:       "Missing closing parenthesis on print statement causes the Python compiler to point to the next line as a SyntaxError.",
		Tags:          []string{"python", "print-statement", "syntax-error"},
		QuestionScore: 1240,
		IsAccepted:    true,
		AnswerBody:    "In Python 3 print is a function. If you forget to close a bracket or quotation mark on the preceding line, the parser will only detect the invalid syntax when attempting to parse the subsequent line.",
		CodeSnippet:   "print(\"Hello world\")\nnext_statement = 42",
	},
	{
		DocID:         11256501,
		Title:         "How to fix Python IndentationError: unexpected indent",
		URL:           "https://stackoverflow.com/questions/11256501",
		Snippet:       "Python uses indentation to define code blocks. An unexpected indent occurs when a line has more leading whitespace or tabs than expected, or when spaces and tabs are mixed.",
		Tags:          []string{"python", "indentation", "pep8"},
		QuestionScore: 2890,
		IsAccepted:    true,
		AnswerBody:    "Ensure consistent indentation across your file. Configure your editor to convert tabs to 4 spaces according to PEP 8 and check for mixed indentation.",
		CodeSnippet:   "def process_data():\n    step1 = True\n    step2 = False",
	},
	{
		DocID:         35593121,
		Title:         "TypeError: 'list' object cannot be interpreted as an integer in range()",
		URL:           "https://stackoverflow.com/questions/35593121",
		Snippet:       "This error happens when passing a list to range() instead of range(len(my_list)) or passing an array into integer arithmetic.",
		Tags:          []string{"python", "list", "typeerror"},
		QuestionScore: 1950,
		IsAccepted:    true,
		AnswerBody:    "range() expects integers as arguments: range(stop) or range(start, stop[, step]). To loop over list indices, use range(len(items)) or iterate over items directly with for item in items.",
		CodeSnippet:   "for i in range(len(items)):\n    print(items[i])",
	},
	{
		DocID:         14872225,
		Title:         "NameError: name 'foobar' is not defined in Python",
		URL:           "https://stackoverflow.com/questions/14872225",
		Snippet:       "A variable, function, or module name was referenced before it was declared, imported, or within an inaccessible scope.",
		Tags:          []string{"python", "scope", "name-error"},
		QuestionScore: 3410,
		IsAccepted:    true,
		AnswerBody:    "Variables must be defined before use. Check spelling of identifiers, verify that module imports succeeded, and ensure variable scope is accessible.",
		CodeSnippet:   "foobar = \"active\"\nprint(foobar)",
	},
	{
		DocID:         10851906,
		Title:         "UnboundLocalError: local variable 'foobar' referenced before assignment",
		URL:           "https://stackoverflow.com/questions/10851906",
		Snippet:       "Occurs inside a function when assigning to a variable that has the same name as an outer/global variable without declaring global.",
		Tags:          []string{"python", "variable-assignment", "scope"},
		QuestionScore: 5120,
		IsAccepted:    true,
		AnswerBody:    "When you assign to a variable inside a function, Python marks it as local to that function. If you read it before the assignment, UnboundLocalError is raised. Use 'global foobar' or pass as parameter.",
		CodeSnippet:   "foobar = 10\ndef increment():\n    global foobar\n    foobar += 1",
	},
	{
		DocID:         4175294,
		Title:         "How to replace a character in a Python string by index? TypeError: 'str' object does not support item assignment",
		URL:           "https://stackoverflow.com/questions/4175294",
		Snippet:       "Strings in Python are immutable objects. You cannot modify an individual character in-place using indexing s[0] = 'x'.",
		Tags:          []string{"python", "string", "immutability", "typeerror"},
		QuestionScore: 3740,
		IsAccepted:    true,
		AnswerBody:    "Python strings cannot be modified in place. Use string slicing and concatenation to build a new string, or convert to a list of characters first.",
		CodeSnippet:   "text = \"H\" + text[1:]",
	},
	{
		DocID:         11041405,
		Title:         "How to handle KeyError exceptions when accessing Python dictionaries",
		URL:           "https://stackoverflow.com/questions/11041405",
		Snippet:       "Raised when trying to access a dictionary key that does not exist. Use the .get(key, default) method or in operator.",
		Tags:          []string{"python", "dictionary", "keyerror"},
		QuestionScore: 2210,
		IsAccepted:    true,
		AnswerBody:    "To prevent KeyError, use dict.get(key, default) or test if key in dict before indexing.",
		CodeSnippet:   "value = data.get(\"spam\", \"default_val\")",
	},
	{
		DocID:         218384,
		Title:         "What is a NullPointerException, and how do I fix it in Java?",
		URL:           "https://stackoverflow.com/questions/218384",
		Snippet:       "A NullPointerException is thrown when an application attempts to use null in a case where an object is required.",
		Tags:          []string{"java", "nullpointerexception", "oop"},
		QuestionScore: 4200,
		IsAccepted:    true,
		AnswerBody:    "NullPointerException occurs when invoking methods or accessing fields on a null object reference. Always initialize variables and check for null or use Optional.",
		CodeSnippet:   "if (bundle != null) {\n    String value = bundle.getString(\"key\");\n}",
	},
	{
		DocID:         19588080,
		Title:         "How to fix IndexOutOfBoundsException in Java ArrayList",
		URL:           "https://stackoverflow.com/questions/19588080",
		Snippet:       "Thrown to indicate that an index of some sort is out of range. Check list.size() before accessing elements.",
		Tags:          []string{"java", "arraylist", "indexoutofboundsexception"},
		QuestionScore: 1650,
		IsAccepted:    true,
		AnswerBody:    "Check bounds before calling list.get(index): ensure index >= 0 && index < list.size().",
		CodeSnippet:   "if (index >= 0 && index < list.size()) {\n    Object item = list.get(index);\n}",
	},
	{
		DocID:         33947834,
		Title:         "How to fix runtime error: invalid memory address or nil pointer dereference in Go",
		URL:           "https://stackoverflow.com/questions/33947834",
		Snippet:       "A nil pointer dereference panic occurs when accessing a field or calling a method on a nil struct pointer or interface.",
		Tags:          []string{"go", "pointers", "panic", "nil"},
		QuestionScore: 2450,
		IsAccepted:    true,
		AnswerBody:    "Always verify pointers are allocated before dereferencing: if ptr != nil { ... }. For structs, allocate with new() or &MyStruct{}.",
		CodeSnippet:   "if ptr != nil {\n    fmt.Println(ptr.Field)\n}",
	},
	{
		DocID:         2346806,
		Title:         "What is a segmentation fault (core dumped) in C++?",
		URL:           "https://stackoverflow.com/questions/2346806",
		Snippet:       "A segmentation fault occurs when a program attempts to access a memory location that it is not allowed to access, such as dereferencing a dangling or null pointer.",
		Tags:          []string{"c++", "segmentation-fault", "pointers", "memory"},
		QuestionScore: 3100,
		IsAccepted:    true,
		AnswerBody:    "Segfaults happen when reading or writing invalid memory: null pointers, uninitialized pointers, out-of-bounds array access, or use-after-free. Use modern smart pointers (std::unique_ptr) and bounds checking.",
		CodeSnippet:   "std::unique_ptr<MyClass> ptr = std::make_unique<MyClass>();\nptr->doWork();",
	},
}

// GetFixture returns the fixture with the specified doc_id.
func GetFixture(docID int64) (Fixture, bool) {
	for _, f := range DefaultFixtures {
		if f.DocID == docID {
			return f, true
		}
	}
	return Fixture{}, false
}

// AllFixtures returns all search fixtures.
func AllFixtures() []Fixture {
	res := make([]Fixture, len(DefaultFixtures))
	copy(res, DefaultFixtures)
	return res
}

// Stub is an in-memory deterministic implementation of SearchClient.
type Stub struct {
	fixtures []Fixture
}

// NewStub constructs a new Stub with default fixtures.
func NewStub() *Stub {
	return &Stub{
		fixtures: DefaultFixtures,
	}
}

// NewStubClient creates a new SearchClient backed by Stub with default fixtures.
func NewStubClient() SearchClient {
	return NewStub()
}

// Tokenize splits a query into lowercase alphanumeric words.
func Tokenize(s string) []string {
	var tokens []string
	var current []rune
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			current = append(current, r)
		} else {
			if len(current) > 0 {
				tokens = append(tokens, string(current))
				current = current[:0]
			}
		}
	}
	if len(current) > 0 {
		tokens = append(tokens, string(current))
	}
	return tokens
}

// Search performs deterministic lexical matching and BM25-like scoring.
func (s *Stub) Search(ctx context.Context, req SearchRequest) (*SearchResponse, error) {
	start := time.Now()

	tokens := Tokenize(req.Query)

	type scoredItem struct {
		fixture Fixture
		score   float64
	}

	var candidates []scoredItem

	for _, fix := range s.fixtures {
		// Tag filtering: if tags provided, keep only threads matching at least one requested tag
		if len(req.Tags) > 0 {
			hasMatchingTag := false
			for _, reqTag := range req.Tags {
				for _, fixTag := range fix.Tags {
					if strings.EqualFold(reqTag, fixTag) {
						hasMatchingTag = true
						break
					}
				}
				if hasMatchingTag {
					break
				}
			}
			if !hasMatchingTag {
				continue
			}
		}

		// Calculate match count
		matchCount := 0
		titleLower := strings.ToLower(fix.Title)
		snippetLower := strings.ToLower(fix.Snippet)

		for _, token := range tokens {
			if strings.Contains(titleLower, token) {
				matchCount += 2
			}
			if strings.Contains(snippetLower, token) {
				matchCount += 1
			}
			for _, tag := range fix.Tags {
				if strings.EqualFold(tag, token) || strings.Contains(strings.ToLower(tag), token) {
					matchCount += 2
					break
				}
			}
		}

		// Only include fixtures with nonzero match count
		if matchCount > 0 {
			// Deterministic score calculation: match count + small tie-breaker from question score
			score := float64(matchCount) + 0.1*(float64(fix.QuestionScore)/1000.0)
			candidates = append(candidates, scoredItem{
				fixture: fix,
				score:   score,
			})
		}
	}

	// Sort candidates by score DESC, then docID ASC for absolute determinism
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].score != candidates[j].score {
			return candidates[i].score > candidates[j].score
		}
		return candidates[i].fixture.DocID < candidates[j].fixture.DocID
	})

	totalHits := int64(len(candidates))

	topK := req.TopK
	if topK <= 0 {
		topK = 10
	} else if topK > 50 {
		topK = 50
	}

	if len(candidates) > topK {
		candidates = candidates[:topK]
	}

	results := make([]Source, len(candidates))
	for i, c := range candidates {
		results[i] = Source{
			DocID:         c.fixture.DocID,
			Title:         c.fixture.Title,
			URL:           c.fixture.URL,
			Snippet:       c.fixture.Snippet,
			Tags:          c.fixture.Tags,
			QuestionScore: c.fixture.QuestionScore,
			BM25Score:     c.score,
			IsAccepted:    c.fixture.IsAccepted,
		}
	}

	tookMs := time.Since(start).Milliseconds()

	return &SearchResponse{
		Query:     req.Query,
		Results:   results,
		TotalHits: totalHits,
		TookMs:    tookMs,
	}, nil
}

// RAG synthesizes an answer and returns relevant code snippet from top match.
func (s *Stub) RAG(ctx context.Context, req SearchRequest) (*RAGResult, error) {
	start := time.Now()

	searchResp, err := s.Search(ctx, req)
	if err != nil {
		return nil, err
	}

	tookMs := time.Since(start).Milliseconds()

	if len(searchResp.Results) == 0 {
		synthesis := fmt.Sprintf("No matching Stack Overflow solutions found for query: %s. Please check for typos or try broader keywords.", req.Query)
		return &RAGResult{
			Query:     req.Query,
			Synthesis: synthesis,
			Code:      nil,
			Sources:   []Source{},
			TookMs:    tookMs,
		}, nil
	}

	topHit := searchResp.Results[0]
	topFix, _ := GetFixture(topHit.DocID)

	synthesis := fmt.Sprintf("Based on Stack Overflow solution for \"%s\", the issue is typically caused by improper syntax or state: %s", topHit.Title, topFix.AnswerBody)
	var code *string
	if topFix.CodeSnippet != "" {
		c := topFix.CodeSnippet
		code = &c
	}

	return &RAGResult{
		Query:     req.Query,
		Synthesis: synthesis,
		Code:      code,
		Sources:   searchResp.Results,
		TookMs:    tookMs,
	}, nil
}
