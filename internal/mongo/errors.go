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

// IsResumePointLost returns true when the change stream cannot be resumed from the
// requested position, typically because it is no longer in the oplog.
func IsResumePointLost(err error) bool {
	var serverErr mongodriver.ServerError
	if !errors.As(err, &serverErr) {
		return false
	}
	return serverErr.HasErrorCode(errorCodeChangeStreamHistoryLost) ||
		serverErr.HasErrorCode(errorCodeChangeStreamFatalError) ||
		serverErr.HasErrorCode(errorCodeInvalidResumeToken)
}
