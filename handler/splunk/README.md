The splunk handler sends events from sawmill to the [Splunk](http://www.splunk.com/) logging service. It works with both [Splunk Enterprise](http://www.splunk.com/en_us/products/splunk-enterprise.html) and [Splunk Cloud](http://www.splunk.com/en_us/products/splunk-cloud.html).

Example Usage:

```go
package main

import (
	"os"

	"github.com/phemmer/sawmill"
	"github.com/phemmer/sawmill/handler/splunk"
)

func main() {
	defer sawmill.Stop()

	splunkURL := "https://username:password@input-prd-p-xl29ahe4v1h3.cloud.splunk.com:8089/?index=development"
	if s, err := splunk.New(splunkURL); err != nil {
		sawmill.Fatal("could not setup splunk handler", sawmill.Fields{"error": err})
	} else {
		sawmill.AddHandler("splunk", s)
	}

	path := "/test"
	_, err := os.Create(path)
	if err != nil {
		sawmill.Error("Failed to create file", sawmill.Fields{"error": err, "path": path})
	}
}
```

![Screenshot](http://i.imgur.com/ZRxSEte.png)
