package mongo

import (
	"errors"

	mongodriver "go.mongodb.org/mongo-driver/v2/mongo"
)

const (
	errorCodeInvalidResumeToken      = 260
	errorCodeChangeStreamFatalError  = 280
	errorCodeChangeStreamHistoryLost = 286
)

// ErrResumePointTooOld is returned when the resume point is older than the configured
// maximum age (e.g. Amazon DocumentDB silently returns no event from an old position)
var ErrResumePointTooOld = errors.New("resume point is older than the configured maximum age")

// IsResumePointLost returns true when the change stream cannot be resumed from the
// requested position, typically because it is no longer in the oplog.
func IsResumePointLost(err error) bool {
	if errors.Is(err, ErrResumePointTooOld) {
		return true
	}
	var serverErr mongodriver.ServerError
	if !errors.As(err, &serverErr) {
		return false
	}
	return serverErr.HasErrorCode(errorCodeChangeStreamHistoryLost) ||
		serverErr.HasErrorCode(errorCodeChangeStreamFatalError) ||
		serverErr.HasErrorCode(errorCodeInvalidResumeToken)
}
