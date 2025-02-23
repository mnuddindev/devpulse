package utils

import (
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/bbalet/stopwords"
)

var stopWords = map[string]bool{
	"a": true, "about": true, "above": true, "after": true, "again": true, "against": true, "all": true, "am": true, "an": true, "and": true, "any": true, "are": true, "aren't": true, "as": true, "at": true,
	"be": true, "because": true, "been": true, "before": true, "being": true, "below": true, "between": true, "both": true, "but": true, "by": true,
	"can't": true, "cannot": true, "could": true, "couldn't": true,
	"did": true, "didn't": true, "do": true, "does": true, "doesn't": true, "doing": true, "don't": true, "down": true, "during": true,
	"each": true,
	"few":  true, "for": true, "from": true, "further": true,
	"had": true, "hadn't": true, "has": true, "hasn't": true, "have": true, "haven't": true, "having": true, "he": true, "he'd": true, "he'll": true, "he's": true, "her": true, "here": true, "here's": true, "hers": true, "herself": true, "him": true, "himself": true, "his": true, "how": true, "how's": true,
	"i": true, "i'd": true, "i'll": true, "i'm": true, "i've": true, "if": true, "in": true, "into": true, "is": true, "isn't": true, "it": true, "it's": true, "its": true, "itself": true,
	"let's": true,
	"me":    true, "more": true, "most": true, "mustn't": true, "my": true, "myself": true,
	"no": true, "nor": true, "not": true, "of": true, "off": true, "on": true, "once": true, "only": true, "or": true, "other": true, "ought": true, "our": true, "ours": true, "ourselves": true, "out": true, "over": true, "own": true,
	"same": true, "shan't": true, "she": true, "she'd": true, "she'll": true, "she's": true, "should": true, "shouldn't": true, "so": true, "some": true, "such": true,
	"than": true, "that": true, "that's": true, "the": true, "their": true, "theirs": true, "them": true, "themselves": true, "then": true, "there": true, "there's": true, "these": true, "they": true, "they'd": true, "they'll": true, "they're": true, "they've": true, "this": true, "those": true, "through": true, "to": true, "too": true,
	"under": true, "until": true, "up": true,
	"very": true,
	"was":  true, "wasn't": true, "we": true, "we'd": true, "we'll": true, "we're": true, "we've": true, "were": true, "weren't": true, "what": true, "what's": true, "when": true, "when's": true, "where": true, "where's": true, "which": true, "while": true, "who": true, "who's": true, "whom": true, "why": true, "why's": true, "with": true, "won't": true, "would": true, "wouldn't": true,
	"you": true, "you'd": true, "you'll": true, "you're": true, "you've": true, "your": true, "yours": true, "yourself": true, "yourselves": true,
}

// Keyword represents a word with its frequency and score
type Keyword struct {
	Word      string
	Frequency int
	Score     float64
}

// GenerateKeywords extracts the best keywords from title and description
func GenerateKeywords(title, description string, maxKeywords int) []string {
	if maxKeywords <= 0 {
		maxKeywords = 5 // Default to 5 keywords
	}

	// Combine title and description, giving title more weight
	text := title + " " + title + " " + description // Title repeated twice for more weight

	// Clean and normalize text
	cleanedText := cleanText(text)

	// Calculate word frequencies
	wordFreq := make(map[string]int)
	words := strings.Fields(cleanedText)
	// totalWords := len(words)

	for _, word := range words {
		if isValidKeyword(word) {
			wordFreq[word]++
		}
	}

	// Calculate keyword scores
	keywords := make([]Keyword, 0, len(wordFreq))
	for word, freq := range wordFreq {
		// Score based on frequency and word length (longer words often more specific)
		score := float64(freq) * (float64(len(word)) / 5.0)
		// Boost score based on position (title words get higher score due to repetition)
		keywords = append(keywords, Keyword{
			Word:      word,
			Frequency: freq,
			Score:     score,
		})
	}

	// Sort keywords by score
	sort.Slice(keywords, func(i, j int) bool {
		if keywords[i].Score == keywords[j].Score {
			// If scores are equal, prefer higher frequency
			return keywords[i].Frequency > keywords[j].Frequency
		}
		return keywords[i].Score > keywords[j].Score
	})

	// Extract top keywords
	result := make([]string, 0, maxKeywords)
	for i := 0; i < len(keywords) && i < maxKeywords; i++ {
		result = append(result, keywords[i].Word)
	}

	return result
}

// cleanText normalizes and cleans the input text
func cleanText(text string) string {
	// Convert to lowercase
	text = strings.ToLower(text)

	// Remove punctuation and special characters
	reg := regexp.MustCompile(`[^a-z0-9\s]`)
	text = reg.ReplaceAllString(text, " ")

	// Remove extra whitespace
	text = strings.Join(strings.Fields(text), " ")

	// Remove stop words using both our list and stopwords package
	text = stopwords.CleanString(text, "en", true)

	return text
}

// isValidKeyword checks if a word should be considered as a keyword
func isValidKeyword(word string) bool {
	// Skip if empty or too short
	if len(word) < 3 {
		return false
	}

	// Skip if it's a stop word
	if stopWords[word] {
		return false
	}

	// Skip if it's just numbers
	isNumber := true
	for _, r := range word {
		if !unicode.IsDigit(r) {
			isNumber = false
			break
		}
	}
	if isNumber {
		return false
	}

	return true
}

// JoinKeywords combines provided keywords with generated ones
func JoinKeywords(provided string, generated []string) string {
	if provided != "" {
		// If keywords are provided, append generated ones only if they're not already present
		existing := strings.Split(strings.ToLower(provided), ",")
		for _, kw := range existing {
			kw = strings.TrimSpace(kw)
			if kw != "" {
				stopWords[kw] = true // Temporarily treat existing keywords as stop words
			}
		}

		uniqueGenerated := make([]string, 0, len(generated))
		for _, gen := range generated {
			if !stopWords[gen] {
				uniqueGenerated = append(uniqueGenerated, gen)
			}
		}

		// Clean up temporary stop words
		for _, kw := range existing {
			delete(stopWords, kw)
		}

		return strings.Join(append(existing, uniqueGenerated...), ", ")
	}

	return strings.Join(generated, ", ")
}
