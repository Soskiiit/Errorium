package search

import "fmt"

// SynthesizeTemplate generates fallback / template RAG synthesis from top search results.
// Used by Stub, GRPCClient, and later LLM orchestrator on degradation.
func SynthesizeTemplate(query string, results []Source) (string, *string) {
	if len(results) == 0 {
		return fmt.Sprintf("No matching Stack Overflow solutions found for query: %s. Please check for typos or try broader keywords.", query), nil
	}

	topHit := results[0]
	topFix, found := GetFixture(topHit.DocID)
	if found {
		synthesis := fmt.Sprintf("Based on Stack Overflow solution for \"%s\", the issue is typically caused by improper syntax or state: %s", topHit.Title, topFix.AnswerBody)
		var code *string
		if topFix.CodeSnippet != "" {
			c := topFix.CodeSnippet
			code = &c
		}
		return synthesis, code
	}

	body := topHit.Snippet
	synthesis := fmt.Sprintf("Based on Stack Overflow solution for \"%s\", the issue is typically caused by improper syntax or state: %s", topHit.Title, body)
	return synthesis, nil
}
