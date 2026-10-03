package drift

import (
	"encoding/json"
	"fmt"
	"sort"
)

// AttributeDiff captures the before and after values of a changed attribute.
type AttributeDiff struct {
	Attribute string      `json:"attribute"`
	Before    interface{} `json:"before,omitempty"`
	After     interface{} `json:"after,omitempty"`
}

// ExtractAttributeDiffs extracts before and after attribute values from a plan Change.
func ExtractAttributeDiffs(change Change) []AttributeDiff {
	changedAttrs := changedAttributesFromChange(change)
	diffs := make([]AttributeDiff, 0, len(changedAttrs))

	for _, attr := range changedAttrs {
		var beforeVal, afterVal interface{}
		if change.Before != nil {
			beforeVal = change.Before[attr]
		}
		if change.After != nil {
			afterVal = change.After[attr]
		}
		if afterVal == nil && change.AfterUnknown != nil && hasUnknownValue(change.AfterUnknown[attr]) {
			afterVal = "(known after apply)"
		}

		diffs = append(diffs, AttributeDiff{
			Attribute: attr,
			Before:    beforeVal,
			After:     afterVal,
		})
	}

	sort.Slice(diffs, func(i, j int) bool {
		return diffs[i].Attribute < diffs[j].Attribute
	})

	return diffs
}

// FormatAttributeValue converts an interface{} attribute value into a concise string.
func FormatAttributeValue(v interface{}) string {
	if v == nil {
		return "null"
	}
	switch typed := v.(type) {
	case string:
		return fmt.Sprintf("%q", typed)
	case int, int64, float64, bool:
		return fmt.Sprintf("%v", typed)
	default:
		data, err := json.Marshal(typed)
		if err != nil {
			return fmt.Sprintf("%v", typed)
		}
		return string(data)
	}
}
