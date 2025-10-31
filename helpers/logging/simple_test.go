package logging

import "testing"

func TestSimpleLogger(t *testing.T) {
	logger := NewLogger("simple")
	logger.Info("test message with arg: %s", "hello")
}
