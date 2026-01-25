package db

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// evaluateSmartSection evaluates rules and returns matching media
func (db *DB) evaluateSmartSection(section *Section, limit, offset int) ([]interface{}, int, error) {
	// Get rules for this section
	rules, err := db.GetSectionRules(section.ID)
	if err != nil {
		return nil, 0, err
	}

	if len(rules) == 0 {
		// No rules, return empty
		return []interface{}{}, 0, nil
	}

	// Build query based on rules
	query, params := buildQueryFromRules(rules, limit, offset)

	// Execute query to get total count
	countQuery := strings.Replace(query, "SELECT *", "SELECT COUNT(*)", 1)
	countQuery = strings.Split(countQuery, "LIMIT")[0] // Remove LIMIT for count

	var total int
	err = db.conn.QueryRow(countQuery, params[:len(params)-2]...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Execute main query
	rows, err := db.conn.Query(query, params...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	// Scan results
	var items []interface{}
	for rows.Next() {
		// This is simplified - in reality you'd need to determine the type
		// and scan into the appropriate struct
		var m Media
		err := rows.Scan(
			&m.ID, &m.Title, &m.OriginalTitle, &m.Type, &m.Year,
			&m.Overview, &m.PosterPath, &m.BackdropPath, &m.Rating, &m.Runtime,
			&m.Genres, &m.TMDbID, &m.IMDbID, &m.SeasonCount, &m.EpisodeCount,
			&m.SourceID, &m.FilePath, &m.FileSize, &m.Duration, &m.VideoCodec,
			&m.AudioCodec, &m.Resolution, &m.AudioTracks, &m.SubtitleTracks,
			&m.CreatedAt, &m.UpdatedAt,
		)
		if err != nil {
			continue
		}
		items = append(items, &m)
	}

	return items, total, rows.Err()
}

// buildQueryFromRules builds a SQL query from section rules
func buildQueryFromRules(rules []SectionRule, limit, offset int) (string, []interface{}) {
	query := "SELECT * FROM media WHERE 1=1"
	params := []interface{}{}

	for _, rule := range rules {
		condition, ruleParams := buildCondition(rule)
		if condition != "" {
			query += " AND " + condition
			params = append(params, ruleParams...)
		}
	}

	query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	params = append(params, limit, offset)

	return query, params
}

// buildCondition builds a SQL condition from a single rule
func buildCondition(rule SectionRule) (string, []interface{}) {
	var condition string
	var params []interface{}

	switch rule.Operator {
	case OperatorEquals:
		// Value should be JSON-encoded, decode it
		var value string
		json.Unmarshal([]byte(rule.Value), &value)
		condition = fmt.Sprintf("%s = ?", rule.Field)
		params = append(params, value)

	case OperatorContains:
		var value string
		json.Unmarshal([]byte(rule.Value), &value)
		condition = fmt.Sprintf("%s LIKE ?", rule.Field)
		params = append(params, "%"+value+"%")

	case OperatorGreaterThan:
		var value float64
		json.Unmarshal([]byte(rule.Value), &value)
		condition = fmt.Sprintf("%s > ?", rule.Field)
		params = append(params, value)

	case OperatorLessThan:
		var value float64
		json.Unmarshal([]byte(rule.Value), &value)
		condition = fmt.Sprintf("%s < ?", rule.Field)
		params = append(params, value)

	case OperatorInRange:
		var values []int
		json.Unmarshal([]byte(rule.Value), &values)
		if len(values) == 2 {
			condition = fmt.Sprintf("%s BETWEEN ? AND ?", rule.Field)
			params = append(params, values[0], values[1])
		}

	case OperatorRegex:
		// SQLite doesn't have native regex, would need extension
		// For now, fall back to LIKE
		var value string
		json.Unmarshal([]byte(rule.Value), &value)
		condition = fmt.Sprintf("%s LIKE ?", rule.Field)
		params = append(params, "%"+value+"%")
	}

	return condition, params
}

// EvaluateMediaAgainstRules checks if a media item matches section rules
func (db *DB) EvaluateMediaAgainstRules(media *Media, rules []SectionRule) bool {
	for _, rule := range rules {
		if !evaluateRule(media, rule) {
			return false // All rules must match (AND logic)
		}
	}
	return true
}

// evaluateRule checks if a single rule matches the media
func evaluateRule(media *Media, rule SectionRule) bool {
	// Get field value from media
	fieldValue := getMediaField(media, rule.Field)

	switch rule.Operator {
	case OperatorEquals:
		var targetValue string
		json.Unmarshal([]byte(rule.Value), &targetValue)
		return fieldValue == targetValue

	case OperatorContains:
		var targetValue string
		json.Unmarshal([]byte(rule.Value), &targetValue)
		return strings.Contains(strings.ToLower(fieldValue), strings.ToLower(targetValue))

	case OperatorGreaterThan:
		var targetValue float64
		json.Unmarshal([]byte(rule.Value), &targetValue)
		numValue, _ := strconv.ParseFloat(fieldValue, 64)
		return numValue > targetValue

	case OperatorLessThan:
		var targetValue float64
		json.Unmarshal([]byte(rule.Value), &targetValue)
		numValue, _ := strconv.ParseFloat(fieldValue, 64)
		return numValue < targetValue

	case OperatorInRange:
		var values []int
		json.Unmarshal([]byte(rule.Value), &values)
		if len(values) == 2 {
			numValue, _ := strconv.Atoi(fieldValue)
			return numValue >= values[0] && numValue <= values[1]
		}

	case OperatorRegex:
		var pattern string
		json.Unmarshal([]byte(rule.Value), &pattern)
		matched, _ := regexp.MatchString(pattern, fieldValue)
		return matched
	}

	return false
}

// getMediaField extracts a field value from media struct
func getMediaField(media *Media, field string) string {
	switch field {
	case "type":
		return string(media.Type)
	case "title":
		return media.Title
	case "year":
		return strconv.Itoa(media.Year)
	case "genre", "genres":
		return media.Genres
	case "rating":
		return fmt.Sprintf("%.1f", media.Rating)
	case "resolution":
		return media.Resolution
	case "video_codec":
		return media.VideoCodec
	case "audio_codec":
		return media.AudioCodec
	default:
		return ""
	}
}

// AutoAssignMediaToSections evaluates all smart sections and assigns media if it matches
func (db *DB) AutoAssignMediaToSections(media *Media) error {
	// Get all smart sections
	sections, err := db.GetAllSections()
	if err != nil {
		return err
	}

	for _, section := range sections {
		if section.SectionType != SectionTypeSmart {
			continue
		}

		// Get rules for this section
		rules, err := db.GetSectionRules(section.ID)
		if err != nil {
			continue
		}

		// Check if media matches all rules
		if db.EvaluateMediaAgainstRules(media, rules) {
			// Add to section
			db.AddMediaToSection(media.ID, media.Type, section.ID)
		}
	}

	return nil
}
