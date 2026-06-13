package cli

import (
	"fmt"
	"strconv"
)

// parseID parses a string as a positive integer id.
func parseID(s string) (int, error) {
	id, err := strconv.Atoi(s)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid id %q: must be a positive integer", s)
	}
	return id, nil
}
