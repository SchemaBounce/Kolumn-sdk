package quarantine

import (
	"crypto/sha1"
	"encoding/hex"
	"regexp"
	"strings"
	"time"
)

const (
	defaultPrefix       = "delete"
	defaultMaxLength    = 255
	minPrefixCharacters = 16
	timeFormat          = "20060102T150405Z"
)

var sanitizeIdentifierPattern = regexp.MustCompile(`[^a-zA-Z0-9_]+`)

// NameOptions controls how a quarantine identifier is generated.
type NameOptions struct {
	Prefix    string
	Kind      string
	Schema    string
	Name      string
	MaxLength int
	Timestamp time.Time
}

// BuildName returns a deterministic, sanitized identifier suitable for quarantine objects.
// It lowercases input, replaces illegal characters, and truncates with a hash suffix when
// the name would exceed the provider's identifier length limit.
func BuildName(opts NameOptions) string {
	prefix := strings.TrimSpace(opts.Prefix)
	if prefix == "" {
		prefix = defaultPrefix
	}

	maxLength := opts.MaxLength
	if maxLength <= 0 {
		maxLength = defaultMaxLength
	}

	ts := opts.Timestamp
	if ts.IsZero() {
		ts = time.Now().UTC()
	}

	parts := []string{prefix}
	if opts.Kind != "" {
		parts = append(parts, opts.Kind)
	}
	parts = append(parts, ts.Format(timeFormat))
	if opts.Schema != "" {
		parts = append(parts, opts.Schema)
	}
	if opts.Name != "" {
		parts = append(parts, opts.Name)
	}

	slug := strings.ToLower(strings.Join(parts, "_"))
	slug = sanitizeIdentifierPattern.ReplaceAllString(slug, "_")
	slug = strings.Trim(slug, "_")
	if slug == "" {
		slug = prefix + "_" + ts.Format(timeFormat)
	}

	if len(slug) <= maxLength {
		return slug
	}

	hash := sha1.Sum([]byte(slug))
	hashSuffix := hex.EncodeToString(hash[:4])

	maxPrefix := maxLength - len(hashSuffix) - 1
	if maxPrefix < minPrefixCharacters {
		maxPrefix = minPrefixCharacters
	}
	if maxPrefix > len(slug) {
		maxPrefix = len(slug)
	}

	prefixSegment := strings.Trim(slug[:maxPrefix], "_")
	if prefixSegment == "" {
		prefixSegment = slug[:maxPrefix]
	}

	return prefixSegment + "_" + hashSuffix
}

// AppendMetadataWithTime stores the quarantine location + timestamp in the provided map.
func AppendMetadataWithTime(state map[string]interface{}, location string, ts time.Time) map[string]interface{} {
	if state == nil {
		state = map[string]interface{}{}
	}

	if location != "" {
		state["quarantine_location"] = location
	}

	if ts.IsZero() {
		ts = time.Now().UTC()
	}
	state["quarantined_at"] = ts.Format(time.RFC3339)

	return state
}

// AppendMetadata uses the current UTC time when recording metadata.
func AppendMetadata(state map[string]interface{}, location string) map[string]interface{} {
	return AppendMetadataWithTime(state, location, time.Time{})
}

// ResolveRelationTarget attempts to determine the schema + object name for a resource using
// the config/state maps, the resource name, and the resource ID. This mirrors the logic used
// by providers when plan data is incomplete.
func ResolveRelationTarget(config, state map[string]interface{}, name, resourceID, fallbackSchema string, nameKeys ...string) (schema, objectName string) {
	schema = stringValue(config, "schema")
	if schema == "" {
		schema = stringValue(state, "schema")
	}

	objectName = firstNonEmpty(config, nameKeys...)
	if objectName == "" {
		objectName = firstNonEmpty(state, append(nameKeys, "name")...)
	}

	candidates := []string{objectName, name, resourceID}
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		if strings.Contains(candidate, ".") {
			parts := strings.SplitN(candidate, ".", 2)
			if schema == "" {
				schema = strings.TrimSpace(parts[0])
			}
			candidate = strings.TrimSpace(parts[1])
		}
		if objectName == "" {
			objectName = candidate
		}
		if schema != "" && objectName != "" {
			break
		}
	}

	if objectName == "" {
		objectName = strings.TrimSpace(name)
	}

	if schema == "" {
		schema = fallbackSchema
	}

	return schema, objectName
}

func firstNonEmpty(m map[string]interface{}, keys ...string) string {
	if m == nil {
		return ""
	}
	for _, key := range keys {
		if key == "" {
			continue
		}
		if val := stringValue(m, key); val != "" {
			return val
		}
	}
	return ""
}

func stringValue(m map[string]interface{}, key string) string {
	if m == nil {
		return ""
	}
	raw, ok := m[key]
	if !ok {
		return ""
	}
	return normalizeString(raw)
}

func normalizeString(value interface{}) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	default:
		return ""
	}
}
