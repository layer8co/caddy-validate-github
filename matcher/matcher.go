package matcher

import (
	"encoding/json"
	"fmt"
	"strings"
)

// matchesPayload returns true when
// 1. every configured JSON path exists in the payload and
// 2. its value matches the expected value.
func MatchesPayload(body []byte, conditions map[string]string) (bool, error) {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return false, fmt.Errorf("decoding failed: %w", err)
	}

	for path, value := range conditions {
		var current any = payload

		for part := range strings.SplitSeq(path, ".") {

			object, ok := current.(map[string]any)
			if !ok {
				return false, nil
			}

			current, ok = object[part]
			if !ok {
				return false, nil
			}
		}

		// The final value must surely be a string and not a map.
		actual, ok := current.(string)
		if !ok || actual != value {
			return false, nil
		}
	}

	// At this point, all the conditions are met.
	return true, nil
}
