package logger

import "go.uber.org/zap/zaptest"

type testingT interface {
	Helper()
	zaptest.TestingT
}

// ForTests wrapped logger for tests.
func ForTests(t testingT) Logger { //nolint:ireturn
	t.Helper()
	return &logger{sugaredLogger: zaptest.NewLogger(t).Sugar()}
}
