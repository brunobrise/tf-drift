package drift

import (
	"errors"
	"fmt"
	"strings"
)

type StateLockError struct {
	Layer   string
	Message string
}

func (e *StateLockError) Error() string {
	return fmt.Sprintf("state lock held on %s: %s", e.Layer, e.Message)
}

func IsStateLockError(err error) bool {
	if err == nil {
		return false
	}
	var lockErr *StateLockError
	return errors.As(err, &lockErr)
}

func isLockErrorOutput(output string) bool {
	lower := strings.ToLower(output)
	return strings.Contains(lower, "error acquiring the state lock") ||
		strings.Contains(lower, "lock info:") ||
		(strings.Contains(lower, "state lock") && strings.Contains(lower, "error"))
}
