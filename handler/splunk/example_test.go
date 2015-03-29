package splunk_test

import (
	"github.com/phemmer/sawmill"
	"github.com/phemmer/sawmill/handler/splunk"
)

const splunkURL = "https://foo:bar@input-prd-p-qrdznsbgja1b.cloud.splunk.com:8089"

func Example() {
	defer sawmill.Stop()

	logger := sawmill.DefaultLogger()

	handler, err := splunk.New(splunkURL)
	if err != nil {
		logger.Fatal("Unable to initialize splunk", sawmill.Fields{"error": err})
	}

	logger.AddHandler("splunk", handler)

	logger.Info("Splunk enabled")
}
