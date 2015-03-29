package filter_test

import (
	"fmt"
	"os"

	// TODO way too many imports for such a simple example
	"github.com/phemmer/sawmill"
	"github.com/phemmer/sawmill/event/formatter"
	"github.com/phemmer/sawmill/handler/filter"
	"github.com/phemmer/sawmill/handler/writer"
)

func Example() {
	logger := sawmill.NewLogger()
	defer logger.Stop()

	writer, err := writer.NewEventWriter(os.Stdout, formatter.SIMPLE_FORMAT)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	handler := filter.New(writer).LevelMin(sawmill.NoticeLevel)
	logger.AddHandler("stdout", handler)

	logger.Debug("This is a debug message")
	logger.Error("This is an error message")

	// Output: This is an error message --
}
