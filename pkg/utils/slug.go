package utils

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/gosimple/slug"
	"github.com/mnuddindev/devpulse/pkg/logger"
	"gorm.io/gorm"
)

// SlugConfig defines customization options for slug generation
type SlugConfig struct {
	SuffixStyle    string // Style of suffix: "numeric" (-2), "version" (-v2), "revision" (-rev2)
	MaxLength      int    // Maximum allowed length of the final slug (e.g., 220 characters)
	MaxAttempts    int    // Maximum number of suffix attempts before fallback (e.g., 100)
	UseCache       bool   // Flag to enable in-memory caching of slug lookups
	FallbackToUUID bool   // Flag to use UUID as a fallback if max attempts are exceeded
}

// DefaultSlugConfig provides sensible defaults for slug configuration
var DefaultSlugConfig = SlugConfig{
	SuffixStyle:    "numeric", // Default to numeric suffixes (e.g., -2)
	MaxLength:      220,       // Default max length matches your CreatePostRequest validation
	MaxAttempts:    100,       // Default to 100 attempts before falling back
	UseCache:       false,     // Default to no caching for simplicity
	FallbackToUUID: true,      // Default to using UUID fallback for robustness
}

// slugCache is an in-memory cache for slug lookups (thread-safe)
var (
	slugCache     = make(map[string]map[string]bool) // Cache maps table+field key to slug existence map
	slugCacheLock sync.RWMutex                       // RWMutex ensures thread-safe cache access
)

// GenerateUniqueSlug generates a unique slug for any model and field with configuration
func GenerateUniqueSlug(db *gorm.DB, model interface{}, fieldName, baseValue string, config ...SlugConfig) (string, error) {
	// Use default config if none provided, otherwise use the first config passed
	cfg := DefaultSlugConfig
	if len(config) > 0 {
		cfg = config[0]
	}

	// Normalize the base value into a URL-safe slug (e.g., "Best SEO" -> "best-seo")
	baseSlug := slug.Make(baseValue)
	// Check if the slug is empty after normalization, which would be invalid
	if baseSlug == "" {
		return "", fmt.Errorf("base value '%s' resulted in an empty slug", baseValue)
	}

	// Calculate max length for base slug, reserving space for suffixes (e.g., -rev100 is ~10 chars)
	maxBaseLength := cfg.MaxLength - 10
	// Truncate base slug if it exceeds the max length minus suffix room
	if len(baseSlug) > maxBaseLength {
		baseSlug = baseSlug[:maxBaseLength]
	}

	// Infer the table name from the model using GORM's naming strategy and reflection
	tableName := db.NamingStrategy.TableName(reflect.TypeOf(model).Elem().Name())
	// Create a unique cache key by combining table name and field name (e.g., "posts:slug")
	cacheKey := tableName + ":" + fieldName

	// Check the in-memory cache if caching is enabled
	if cfg.UseCache {
		// Acquire a read lock for safe concurrent access to the cache
		slugCacheLock.RLock()
		// Check if the cache has data for this table+field combination
		if cachedSlugs, exists := slugCache[cacheKey]; exists {
			// If the base slug isn’t in the cache, it’s available—return it
			if !cachedSlugs[baseSlug] {
				// Release the read lock before returning
				slugCacheLock.RUnlock()
				return baseSlug, nil
			}
		}
		// Release the read lock if no early return
		slugCacheLock.RUnlock()
	}

	// Prepare a slice to store existing slug values from the database
	var existingValues []string
	// Build a SQL query to fetch all values of the field that start with baseSlug (e.g., "slug LIKE 'best-seo%'")
	query := fmt.Sprintf("%s LIKE ?", fieldName)
	// Execute the query to pluck the field values into existingValues
	if err := db.Model(model).
		Where(query, baseSlug+"%").
		Pluck(fieldName, &existingValues).Error; err != nil {
		// Log the error with context for debugging
		logger.Log.WithError(err).Errorf("Failed to fetch existing %s values for %s in table %s", fieldName, baseSlug, tableName)
		// Return a detailed error message including table, field, and slug
		return "", fmt.Errorf("failed to fetch existing %s values for %s in table %s: %w", fieldName, baseSlug, tableName, err)
	}

	// Update the cache with fetched values if caching is enabled
	if cfg.UseCache {
		// Acquire a write lock for safe cache modification
		slugCacheLock.Lock()
		// Initialize the cache entry for this table+field if it doesn’t exist
		if _, exists := slugCache[cacheKey]; !exists {
			slugCache[cacheKey] = make(map[string]bool)
		}
		// Mark each existing value as taken in the cache
		for _, val := range existingValues {
			slugCache[cacheKey][val] = true
		}
		// Release the write lock
		slugCacheLock.Unlock()
	}

	// Create a map to track used suffix numbers (e.g., 0 for no suffix, 2 for -2)
	suffixMap := make(map[int]bool)
	// Compile a regex to match baseSlug with optional suffixes (e.g., "best-seo-2" or "best-seo-v3")
	re := regexp.MustCompile(`^` + regexp.QuoteMeta(baseSlug) + `(?:-([a-zA-Z0-9]+))?$`)
	// Iterate over existing values to populate suffixMap
	for _, value := range existingValues {
		// Try to match the value against the regex
		matches := re.FindStringSubmatch(value)
		// Check if there’s a match with a suffix
		if len(matches) > 1 && matches[1] != "" {
			// Handle different suffix styles
			switch cfg.SuffixStyle {
			case "numeric":
				// Convert numeric suffix to integer (e.g., "2" from "-2")
				if num, err := strconv.Atoi(matches[1]); err == nil {
					suffixMap[num] = true
				}
			case "version":
				// Check for "v" prefix and convert number (e.g., "v3" -> 3)
				if strings.HasPrefix(matches[1], "v") {
					if num, err := strconv.Atoi(matches[1][1:]); err == nil {
						suffixMap[num] = true
					}
				}
			case "revision":
				// Check for "rev" prefix and convert number (e.g., "rev4" -> 4)
				if strings.HasPrefix(matches[1], "rev") {
					if num, err := strconv.Atoi(matches[1][3:]); err == nil {
						suffixMap[num] = true
					}
				}
			}
			// If no suffix and matches baseSlug exactly, mark 0 as taken
		} else if value == baseSlug {
			suffixMap[0] = true
		}
	}

	// Check if the base slug (no suffix) is available
	if !suffixMap[0] {
		// If caching, mark the base slug as taken
		if cfg.UseCache {
			slugCacheLock.Lock()
			slugCache[cacheKey][baseSlug] = true
			slugCacheLock.Unlock()
		}
		// Return the base slug since it’s free
		return baseSlug, nil
	}

	// Try suffixes from 2 up to MaxAttempts
	for i := 2; i <= cfg.MaxAttempts; i++ {
		// Declare candidate variable to hold the generated slug
		var candidate string
		// Generate the candidate slug based on the suffix style
		switch cfg.SuffixStyle {
		case "numeric":
			candidate = fmt.Sprintf("%s-%d", baseSlug, i)
		case "version":
			candidate = fmt.Sprintf("%s-v%d", baseSlug, i)
		case "revision":
			candidate = fmt.Sprintf("%s-rev%d", baseSlug, i)
		default:
			// Return an error for unsupported suffix styles
			return "", fmt.Errorf("unsupported suffix style: %s", cfg.SuffixStyle)
		}

		// Skip this candidate if it exceeds the max length
		if len(candidate) > cfg.MaxLength {
			continue
		}

		// Check if this suffix number is unused
		if !suffixMap[i] {
			// If caching, mark the candidate as taken
			if cfg.UseCache {
				slugCacheLock.Lock()
				slugCache[cacheKey][candidate] = true
				slugCacheLock.Unlock()
			}
			// Return the candidate since it’s unique
			return candidate, nil
		}
	}

	// If configured, attempt a UUID fallback
	if cfg.FallbackToUUID {
		// Generate a candidate with a UUID suffix
		candidate := fmt.Sprintf("%s-%s", baseSlug, uuid.New().String()[:8])
		// Check if the candidate fits within max length
		if len(candidate) <= cfg.MaxLength {
			// If caching, mark the candidate as taken
			if cfg.UseCache {
				slugCacheLock.Lock()
				slugCache[cacheKey][candidate] = true
				slugCacheLock.Unlock()
			}
			// Return the UUID-suffixed slug
			return candidate, nil
		}
	}

	// Return an error if no unique slug was found within limits
	return "", fmt.Errorf("could not generate unique %s for %s within %d attempts", fieldName, baseSlug, cfg.MaxAttempts)
}
