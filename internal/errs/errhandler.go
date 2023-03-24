package errs

import (
	"strings"
)

func ErrorHandlerBotBlocked(err error) bool {

	lookFor := "Forbidden: bot was blocked by the user"
	contain := strings.Contains(err.Error(), lookFor)

	switch contain {
	case true:
		return contain
	}

	return contain

}
