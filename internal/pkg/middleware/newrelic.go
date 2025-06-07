package middleware

import (
	"context"

	"github.com/labstack/echo/v4"
	"github.com/newrelic/go-agent/v3/newrelic"
)

// AddAttribute adds a custom attribute to the current transaction
func AddAttribute(c echo.Context, key string, value interface{}) {
	if txn := newrelic.FromContext(c.Request().Context()); txn != nil {
		txn.AddAttribute(key, value)
	}
}

// NoticeError reports an error to New Relic
func NoticeError(c echo.Context, err error) {
	if txn := newrelic.FromContext(c.Request().Context()); txn != nil {
		txn.NoticeError(err)
	}
}

// SetUserID sets the user ID attribute for the current transaction
func SetUserID(c echo.Context, userID string) {
	AddAttribute(c, "user.id", userID)
}

// SetMatchID sets the match ID attribute for the current transaction
func SetMatchID(c echo.Context, matchID string) {
	AddAttribute(c, "match.id", matchID)
}

// Context returns the context from the Echo context, which includes New Relic transaction context
func Context(c echo.Context) context.Context {
	return c.Request().Context()
}

// SetTransactionName sets the transaction name for better tracing visibility
func SetTransactionName(ctx context.Context, name string) {
	txn := newrelic.FromContext(ctx)
	if txn != nil {
		txn.SetName(name)
	}
}
