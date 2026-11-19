package graph

import (
	"context"
	"math"
	"sort"
	"strings"

	"github.com/soskiiit/errorium/services/gateway/internal/search"
)

// PoolThread represents a Stack Overflow thread in the related-thread pool.
type PoolThread struct {
	DocID    int64
	Title    string
	Snippet  string
	Tags     []string
	Score    int
	Accepted bool
	Answer   string
	URL      string
}

// DefaultPool contains 14 realistic Stack Overflow related threads from the knowledge base.
var DefaultPool = []PoolThread{
	{
		DocID:    1132941,
		Title:    "Least Astonishment and the Mutable Default Argument",
		Snippet:  "Why does a Python function with a default list argument keep values between calls? The default object is created once at definition time.",
		Tags:     []string{"python", "list", "default-argument"},
		Score:    6890,
		Accepted: true,
		Answer:   "Default arguments are evaluated once, when the function is defined — not on each call. A mutable default (list, dict) is therefore shared across all calls.\n\nUse None as the default and create the object inside the function:\n\ndef append_to(item, target=None):\n    if target is None:\n        target = []\n    target.append(item)\n    return target",
		URL:      "https://stackoverflow.com/questions/1132941",
	},
	{
		DocID:    53513,
		Title:    "How do I check if a list is empty?",
		Snippet:  "What is the most pythonic way to test whether a list has no elements? Comparing to [] or checking len() both feel verbose.",
		Tags:     []string{"python", "list"},
		Score:    4520,
		Accepted: true,
		Answer:   "Empty sequences are falsy in Python, so the idiomatic check is:\n\nif not my_list:\n    print(\"list is empty\")\n\nAvoid len(my_list) == 0 — it is slower and less readable. The same pattern works for strings, tuples and dicts.",
		URL:      "https://stackoverflow.com/questions/53513",
	},
	{
		DocID:    1720421,
		Title:    "How to concatenate two lists in Python?",
		Snippet:  "I want to merge list_a and list_b into a single list. What is the difference between +, extend() and unpacking?",
		Tags:     []string{"python", "list", "concatenation"},
		Score:    3110,
		Accepted: true,
		Answer:   "Use + to build a new list, or extend() to mutate in place:\n\nmerged = list_a + list_b      # new list\nlist_a.extend(list_b)         # modifies list_a\n\nSince Python 3.5 you can also unpack: merged = [*list_a, *list_b]. Note that + requires both operands to be lists, while extend() accepts any iterable.",
		URL:      "https://stackoverflow.com/questions/1720421",
	},
	{
		DocID:    38987,
		Title:    "How do I merge two dictionaries in a single expression?",
		Snippet:  "Merging dict x and dict y so that values from y win on key conflicts. Looking for a clean one-liner.",
		Tags:     []string{"python", "dictionary"},
		Score:    4860,
		Accepted: true,
		Answer:   "Python 3.9+ supports the merge operator:\n\nz = x | y\n\nPython 3.5+: z = {**x, **y}\n\nOlder versions: z = dict(x, **y) — only works when y keys are strings. In all cases values from y override values from x on conflicts.",
		URL:      "https://stackoverflow.com/questions/38987",
	},
	{
		DocID:    176918,
		Title:    "Finding the index of an item in a list",
		Snippet:  "Given a list and a value, how do I get the position of the value? And what happens when the value is not present?",
		Tags:     []string{"python", "list", "indexing"},
		Score:    3720,
		Accepted: true,
		Answer:   "Use the built-in index() method:\n\nidx = my_list.index(\"foo\")\n\nIt raises ValueError when the item is absent, so guard with a membership test or try/except. To get all positions use enumerate:\n\n[i for i, v in enumerate(my_list) if v == \"foo\"]",
		URL:      "https://stackoverflow.com/questions/176918",
	},
	{
		DocID:    231767,
		Title:    "What does the 'yield' keyword do in Python?",
		Snippet:  "Confused about generators: how does yield differ from return, and what is the point of iterating over a generator?",
		Tags:     []string{"python", "generator", "yield"},
		Score:    7340,
		Accepted: true,
		Answer:   "yield turns a function into a generator. Each call to next() runs the function until the next yield, produces that value and suspends state — instead of computing everything upfront like return.\n\ndef countdown(n):\n    while n > 0:\n        yield n\n        n -= 1\n\nGenerators are lazy and memory-efficient: they produce items one at a time, which matters for large or infinite sequences.",
		URL:      "https://stackoverflow.com/questions/231767",
	},
	{
		DocID:    2052390,
		Title:    "Manually raising (throwing) an exception in Python",
		Snippet:  "How do I raise an exception on purpose so that callers can catch it? Which exception type should I pick?",
		Tags:     []string{"python", "exception", "raise"},
		Score:    3280,
		Accepted: true,
		Answer:   "Use the raise statement with an exception instance:\n\nraise ValueError(\"config must not be empty\")\n\nPick the built-in type that matches the semantics (ValueError, TypeError, KeyError...). Inside an except block, bare raise re-raises the current exception preserving the traceback. Define custom types by subclassing Exception for domain errors.",
		URL:      "https://stackoverflow.com/questions/2052390",
	},
	{
		DocID:    1712227,
		Title:    "How do I get the number of elements in a list in Python?",
		Snippet:  "Looking for the equivalent of length/size methods from other languages to count items in a list.",
		Tags:     []string{"python", "list", "len"},
		Score:    4210,
		Accepted: true,
		Answer:   "Use the built-in len() function:\n\ncount = len(my_list)\n\nlen() is O(1) for lists — the size is stored, not counted. It works on any sized container: str, tuple, dict, set. For lazy iterators without a known size, consume them with sum(1 for _ in it) instead.",
		URL:      "https://stackoverflow.com/questions/1712227",
	},
	{
		DocID:    613183,
		Title:    "How do I sort a dictionary by value?",
		Snippet:  "I have a dict of word counts and want to order entries by value descending. dict itself has no sort method.",
		Tags:     []string{"python", "dictionary", "sorting"},
		Score:    3540,
		Accepted: true,
		Answer:   "Sort the items with a key function:\n\nsorted_items = sorted(d.items(), key=lambda kv: kv[1], reverse=True)\n\nSince Python 3.7 dicts preserve insertion order, so you can rebuild one: dict(sorted_items). For the top-n entries, collections.Counter(d).most_common(n) is even shorter.",
		URL:      "https://stackoverflow.com/questions/613183",
	},
	{
		DocID:    101268,
		Title:    "Hidden features of Python",
		Snippet:  "Community wiki of lesser-known language features: chained comparisons, argument unpacking, else on loops and more.",
		Tags:     []string{"python", "language-features"},
		Score:    5230,
		Accepted: true,
		Answer:   "A few favorites:\n\n- Chained comparisons: 0 < x < 10\n- Argument unpacking: func(*args, **kwargs)\n- for/else: else runs when the loop was not broken\n- Conditional expressions: value = a if cond else b\n- Swapping without a temp: a, b = b, a\n\nMost of these remove boilerplate while keeping the code readable.",
		URL:      "https://stackoverflow.com/questions/101268",
	},
	{
		DocID:    26595,
		Title:    "Is there any difference between 'foo is None' and 'foo == None'?",
		Snippet:  "Both checks seem to work. Is one of them preferred, and can == None ever give a wrong answer?",
		Tags:     []string{"python", "none", "comparison"},
		Score:    1620,
		Accepted: true,
		Answer:   "Use is None. None is a singleton, and identity comparison is exact and fast. == invokes __eq__, which a class can overload to return True for anything — so == None can lie. PEP 8 explicitly recommends is / is not for None checks.",
		URL:      "https://stackoverflow.com/questions/26595",
	},
	{
		DocID:    13076951,
		Title:    "TypeError: 'int' object is not iterable when looping",
		Snippet:  "for i in 5: raises TypeError. I expected it to loop five times like range-based loops in other languages.",
		Tags:     []string{"python", "typeerror", "list"},
		Score:    1240,
		Accepted: true,
		Answer:   "An int is not iterable — Python has no implicit numeric range. Wrap the number in range():\n\nfor i in range(5):\n    print(i)\n\nThe same TypeError appears when unpacking or passing an int where a sequence is expected, e.g. list(5). Check that the object you iterate over is actually a collection.",
		URL:      "https://stackoverflow.com/questions/13076951",
	},
	{
		DocID:    419163,
		Title:    "What does if __name__ == '__main__': do?",
		Snippet:  "Every script seems to end with this guard. What problem does it solve when a file is both imported and executed?",
		Tags:     []string{"python", "module", "main"},
		Score:    4930,
		Accepted: true,
		Answer:   "When a file runs as a script, Python sets __name__ to \"__main__\"; when it is imported, __name__ is the module name. The guard therefore runs code only on direct execution:\n\nif __name__ == \"__main__\":\n    main()\n\nThis lets a module expose reusable functions without side effects on import — essential for testable code.",
		URL:      "https://stackoverflow.com/questions/419163",
	},
	{
		DocID:    379906,
		Title:    "How do I convert a string to an integer in Python?",
		Snippet:  "Parsing numeric user input: int(\"42\") works, but what about floats, whitespace and invalid strings?",
		Tags:     []string{"python", "string", "type-conversion"},
		Score:    2860,
		Accepted: true,
		Answer:   "Use int() for whole numbers and float() for decimals:\n\nn = int(\"42\")        # 42\nx = float(\"3.14\")    # 3.14\n\nint() tolerates surrounding whitespace but raises ValueError on invalid input (\"4.2\" is not an int string — go through float first). Wrap parsing in try/except ValueError when the source is user input.",
		URL:      "https://stackoverflow.com/questions/379906",
	},
}

// Stub is an in-memory deterministic implementation of GraphClient.
type Stub struct {
	pool []PoolThread
}

// NewStub constructs a new GraphClient stub with default related-thread pool.
func NewStub() *Stub {
	return &Stub{
		pool: DefaultPool,
	}
}

func (s *Stub) getPoolThread(docID int64) (PoolThread, bool) {
	for _, p := range s.pool {
		if p.DocID == docID {
			return p, true
		}
	}
	return PoolThread{}, false
}

// GetThreadGraph constructs a Star Map graph from query docIDs and related pool threads.
func (s *Stub) GetThreadGraph(ctx context.Context, docIDs []int64) (*ThreadGraph, error) {
	usedIDs := make(map[int64]bool)
	var nodes []ThreadNode

	// 1. Process query docIDs: first valid is PRIMARY_RESULT, subsequent are OTHER_RESULT.
	for _, id := range docIDs {
		if usedIDs[id] {
			continue
		}
		fixture, found := search.GetFixture(id)
		if !found {
			continue
		}

		category := "OTHER_RESULT"
		if len(nodes) == 0 {
			category = "PRIMARY_RESULT"
		}

		bestScore := int(math.Max(1, float64(fixture.QuestionScore)*0.4))
		node := ThreadNode{
			DocID:             fixture.DocID,
			Category:          category,
			Score:             float64(fixture.QuestionScore),
			Title:             fixture.Title,
			Snippet:           fixture.Snippet,
			HasAcceptedAnswer: fixture.IsAccepted,
			Tags:              fixture.Tags,
			BestAnswerScore:   bestScore,
			BestAnswerBody:    fixture.AnswerBody,
			OriginalURL:       fixture.URL,
		}
		nodes = append(nodes, node)
		usedIDs[id] = true
	}

	// If no valid query nodes found, return empty graph
	if len(nodes) == 0 {
		return &ThreadGraph{
			Nodes: []ThreadNode{},
			Edges: []ThreadEdge{},
		}, nil
	}

	// 2. Collect tags from query nodes for relevance calculation
	queryTags := make(map[string]bool)
	for _, node := range nodes {
		for _, tag := range node.Tags {
			queryTags[strings.ToLower(tag)] = true
		}
	}

	// 3. Score pool candidates by tag overlap
	type candidate struct {
		thread  PoolThread
		overlap int
	}

	var candidates []candidate
	for _, p := range s.pool {
		if usedIDs[p.DocID] {
			continue
		}
		overlap := 0
		for _, tag := range p.Tags {
			if queryTags[strings.ToLower(tag)] {
				overlap++
			}
		}
		candidates = append(candidates, candidate{
			thread:  p,
			overlap: overlap,
		})
	}

	// Deterministic sort: overlap DESC, score DESC, docID ASC
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].overlap != candidates[j].overlap {
			return candidates[i].overlap > candidates[j].overlap
		}
		if candidates[i].thread.Score != candidates[j].thread.Score {
			return candidates[i].thread.Score > candidates[j].thread.Score
		}
		return candidates[i].thread.DocID < candidates[j].thread.DocID
	})

	// Take up to 12 related candidates
	maxRelated := 12
	if len(candidates) > maxRelated {
		candidates = candidates[:maxRelated]
	}

	for i, c := range candidates {
		category := "LINKED"
		if i < 3 {
			category = "DUPLICATE"
		}

		bestScore := int(math.Max(1, float64(c.thread.Score)*0.6))
		node := ThreadNode{
			DocID:             c.thread.DocID,
			Category:          category,
			Score:             float64(c.thread.Score),
			Title:             c.thread.Title,
			Snippet:           c.thread.Snippet,
			HasAcceptedAnswer: c.thread.Accepted,
			Tags:              c.thread.Tags,
			BestAnswerScore:   bestScore,
			BestAnswerBody:    c.thread.Answer,
			OriginalURL:       c.thread.URL,
		}
		nodes = append(nodes, node)
		usedIDs[c.thread.DocID] = true
	}

	// 4. Build directed edges without self-loops
	var primary *ThreadNode
	var others []ThreadNode
	var duplicates []ThreadNode
	var linked []ThreadNode

	for i := range nodes {
		switch nodes[i].Category {
		case "PRIMARY_RESULT":
			if primary == nil {
				primary = &nodes[i]
			}
		case "OTHER_RESULT":
			others = append(others, nodes[i])
		case "DUPLICATE":
			duplicates = append(duplicates, nodes[i])
		case "LINKED":
			linked = append(linked, nodes[i])
		}
	}

	var edges []ThreadEdge

	if primary != nil {
		// OTHER -> PRIMARY (LINKED_IN_ANSWER)
		for _, o := range others {
			if o.DocID != primary.DocID {
				edges = append(edges, ThreadEdge{
					SourceDocID: o.DocID,
					TargetDocID: primary.DocID,
					Type:        "LINKED_IN_ANSWER",
				})
			}
		}

		// DUPLICATE -> PRIMARY (DUPLICATE_OF)
		for _, d := range duplicates {
			if d.DocID != primary.DocID {
				edges = append(edges, ThreadEdge{
					SourceDocID: d.DocID,
					TargetDocID: primary.DocID,
					Type:        "DUPLICATE_OF",
				})
			}
		}

		// LINKED -> PRIMARY or OTHER (LINKED_IN_ANSWER)
		for i, l := range linked {
			targetID := primary.DocID
			if len(others) > 0 && i%2 == 1 {
				targetID = others[(i/2)%len(others)].DocID
			}
			if l.DocID != targetID {
				edges = append(edges, ThreadEdge{
					SourceDocID: l.DocID,
					TargetDocID: targetID,
					Type:        "LINKED_IN_ANSWER",
				})
			}
		}
	}

	return &ThreadGraph{
		Nodes: nodes,
		Edges: edges,
	}, nil
}

// GetThreadDetails looks up a thread in search fixtures or graph pool.
func (s *Stub) GetThreadDetails(ctx context.Context, docID int64) (*ThreadDetails, error) {
	// First check search fixtures
	if f, ok := search.GetFixture(docID); ok {
		return &ThreadDetails{
			DocID:             f.DocID,
			Title:             f.Title,
			BestAnswerBody:    f.AnswerBody,
			OriginalURL:       f.URL,
			Tags:              f.Tags,
			QuestionScore:     f.QuestionScore,
			HasAcceptedAnswer: f.IsAccepted,
		}, nil
	}

	// Then check graph pool
	if p, ok := s.getPoolThread(docID); ok {
		return &ThreadDetails{
			DocID:             p.DocID,
			Title:             p.Title,
			BestAnswerBody:    p.Answer,
			OriginalURL:       p.URL,
			Tags:              p.Tags,
			QuestionScore:     p.Score,
			HasAcceptedAnswer: p.Accepted,
		}, nil
	}

	return nil, ErrNotFound
}
