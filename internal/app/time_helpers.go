package app

import (
	"time"

	"github.com/gin-gonic/gin"
)

func parseOptionalRFC3339(value *string) (*time.Time, error) {
	if value == nil || *value == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, *value)
	if err != nil {
		return nil, err
	}
	utc := parsed.UTC()
	return &utc, nil
}

func formatAPITime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func writeStaleUpdate(c *gin.Context, currentUpdatedAt time.Time) {
	c.JSON(409, gin.H{
		"error":              "stale update: current record is newer",
		"current_updated_at": formatAPITime(currentUpdatedAt),
	})
}
