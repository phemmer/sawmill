package sawmill

import (
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/phemmer/sawmill/event"
)

func TestNewWriter(t *testing.T) {
	handler := NewChannelHandler()
	logger := NewLogger()
	defer logger.Stop()
	logger.AddHandler("TestNewWriter", handler)

	writer := logger.NewWriter(-1)

	writer.Write([]byte("TestNewWriter"))
	runtime.Gosched()
	writer.Write([]byte(" part 2"))
	runtime.Gosched()
	writer.Write([]byte{'\n'})

	logEvent := handler.Next(time.Second)

	if assert.NotNil(t, logEvent) {
		assert.Equal(t, event.Info, logEvent.Level)
		assert.Equal(t, "TestNewWriter part 2", logEvent.Message)
	}
}
