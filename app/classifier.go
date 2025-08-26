package app

import "strings"

// ClassifyPrompt analyzes a user input and returns a category for metrics tracking
// This centralizes the prompt classification logic used by both CLI and Web modes
func ClassifyPrompt(input string) string {
	if input == "" {
		return "general"
	}

	lowerInput := strings.ToLower(input)

	// Use switch for comprehensive keyword matching
	switch {
	case strings.Contains(lowerInput, "code") || strings.Contains(lowerInput, "debug") ||
		strings.Contains(lowerInput, "programming") || strings.Contains(lowerInput, "function") ||
		strings.Contains(lowerInput, "variable") || strings.Contains(lowerInput, "syntax") ||
		strings.Contains(lowerInput, "compile") || strings.Contains(lowerInput, "error") ||
		strings.Contains(lowerInput, "algorithm") || strings.Contains(lowerInput, "script"):
		return "code_help"

	case strings.Contains(lowerInput, "explain") || strings.Contains(lowerInput, "how") ||
		strings.Contains(lowerInput, "what") || strings.Contains(lowerInput, "why") ||
		strings.Contains(lowerInput, "define") || strings.Contains(lowerInput, "describe") ||
		strings.Contains(lowerInput, "clarify") || strings.Contains(lowerInput, "understand") ||
		strings.Contains(lowerInput, "meaning") || strings.Contains(lowerInput, "difference"):
		return "explanation"

	case strings.Contains(lowerInput, "write") || strings.Contains(lowerInput, "create") ||
		strings.Contains(lowerInput, "generate") || strings.Contains(lowerInput, "compose") ||
		strings.Contains(lowerInput, "draft") || strings.Contains(lowerInput, "make") ||
		strings.Contains(lowerInput, "build") || strings.Contains(lowerInput, "design") ||
		strings.Contains(lowerInput, "story") || strings.Contains(lowerInput, "poem"):
		return "creative"

	case strings.Contains(lowerInput, "analyze") || strings.Contains(lowerInput, "review") ||
		strings.Contains(lowerInput, "compare") || strings.Contains(lowerInput, "evaluate") ||
		strings.Contains(lowerInput, "assess") || strings.Contains(lowerInput, "critique") ||
		strings.Contains(lowerInput, "examine") || strings.Contains(lowerInput, "study"):
		return "analysis"

	case strings.Contains(lowerInput, "solve") || strings.Contains(lowerInput, "calculate") ||
		strings.Contains(lowerInput, "math") || strings.Contains(lowerInput, "equation") ||
		strings.Contains(lowerInput, "formula") || strings.Contains(lowerInput, "problem") ||
		strings.Contains(lowerInput, "compute") || strings.Contains(lowerInput, "number"):
		return "problem_solving"

	case strings.Contains(lowerInput, "translate") || strings.Contains(lowerInput, "language") ||
		strings.Contains(lowerInput, "grammar") || strings.Contains(lowerInput, "spell") ||
		strings.Contains(lowerInput, "correct") || strings.Contains(lowerInput, "edit"):
		return "language"

	default:
		return "general"
	}
}
