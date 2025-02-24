package utils

import (
	"regexp"
	"sort"
	"strings"
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

// MetaOutput for response
type MetaOutput struct {
	MetaTitle string   `json:"meta_title"`
	MetaDesc  string   `json:"meta_desc"`
	Keywords  []string `json:"keywords"`
}

// cleanText normalizes text
func cleanText(text string) string {
	text = strings.ToLower(text)
	reg := regexp.MustCompile(`[^a-z0-9\s]`)
	return strings.TrimSpace(reg.ReplaceAllString(text, " "))
}

// generateKeywords extracts SEO-optimized keywords
func GenerateKeywords(title, content string, maxKeywords int) []string {
	if maxKeywords <= 0 {
		maxKeywords = 5
	}

	// Combine text, weighting title heavily
	fullText := title + " " + title + " " + title + " " + content // Title 3x for SEO priority
	cleaned := cleanText(fullText)
	words := strings.Fields(cleaned)

	// Build n-grams (1-3 words)
	ngrams := make(map[string]float64)
	totalWords := float64(len(words))
	for i := 0; i < len(words); i++ {
		for n := 1; n <= 3 && i+n <= len(words); n++ {
			phrase := strings.Join(words[i:i+n], " ")
			if len(phrase) < 3 || stopWords[phrase] {
				continue
			}
			ngrams[phrase]++
		}
	}

	// Score n-grams
	type kw struct {
		Word  string
		Score float64
	}
	kws := make([]kw, 0, len(ngrams))
	for phrase, freq := range ngrams {
		score := freq / totalWords // Base density
		wordCount := float64(len(strings.Fields(phrase)))
		score *= (1.0 + wordCount*0.3) // Boost multi-word phrases
		if strings.Contains(strings.ToLower(title), phrase) {
			score *= 3.0 // Heavy title boost (SEO expert tactic)
		}
		if freq > 1 { // Favor repeated terms
			score *= 1.5
		}
		kws = append(kws, kw{phrase, score})
	}

	// Sort by score
	sort.Slice(kws, func(i, j int) bool {
		if kws[i].Score == kws[j].Score {
			return len(kws[i].Word) > len(kws[j].Word) // Tiebreaker: longer phrases
		}
		return kws[i].Score > kws[j].Score
	})

	// Select top keywords
	result := make([]string, 0, maxKeywords)
	for i := 0; i < len(kws) && i < maxKeywords; i++ {
		result = append(result, kws[i].Word)
	}
	return result
}

// generateMeta crafts SEO-optimized metadata
func GenerateMeta(title, content, brand string) MetaOutput {
	keywords := GenerateKeywords(title, content, 10)
	primaryKW := keywords[0] // Top keyword for title/desc

	// Craft Meta Title (SEO expert style: keyword + benefit + brand)
	metaTitle := primaryKW + " Guide for Success"
	if brand != "" {
		metaTitle += " | " + brand
	}
	metaTitle = TruncateString(metaTitle, 60)

	// Craft Meta Description (SEO expert style: hook + keyword + CTA)
	metaDesc := "Discover " + primaryKW + " to skyrocket your results in " + brand + ". Click now!"
	metaDesc = TruncateString(metaDesc, 160)

	return MetaOutput{
		MetaTitle: metaTitle,
		MetaDesc:  metaDesc,
		Keywords:  keywords,
	}
}

// JoinKeywords combines provided and generated keywords
func JoinKeywords(provided string, generated []string) string {
	if provided != "" {
		existing := strings.Split(strings.ToLower(provided), ",")
		for _, kw := range existing {
			kw = strings.TrimSpace(kw)
			if kw != "" {
				stopWords[kw] = true
			}
		}
		uniqueGenerated := make([]string, 0, len(generated))
		for _, gen := range generated {
			if !stopWords[gen] {
				uniqueGenerated = append(uniqueGenerated, gen)
			}
		}
		for _, kw := range existing {
			delete(stopWords, kw)
		}
		return strings.Join(append(existing, uniqueGenerated...), ", ")
	}
	return strings.Join(generated, ", ")
}
