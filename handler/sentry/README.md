The sentry handler provides a handler for sawmill which sends events to the [Sentry](http://getsentry.com) service.

Example usage:

```go
package main

import (
	"os"
	"strings"

	"github.com/phemmer/sawmill"
	"github.com/phemmer/sawmill/handler/sentry"
)

func main() {
	defer sawmill.Stop()
	sawmill.SetStackMinLevel(sawmill.ErrorLevel)

	sentryDSN := "https://00112233445566778899aabbccddeeff:0123456789abcdef0123456789abcdef@app.getsentry.com/12345"
	if s, err := sentry.New(sentryDSN); err == nil {
		filter := sawmill.FilterHandler(s).LevelMin(sawmill.ErrorLevel)
		sawmill.AddHandler("sentry", filter)
	}

	_, err := os.Create("/test")
	if err != nil {
		sawmill.Error("Failed to create /test", sawmill.Fields{"error": err, "path": "/test"})
	}
}
```

---

### Stream
![Stream](http://i.imgur.com/AEMBOMM.png)

---
### Sample
![Sample](http://i.imgur.com/TsNFXgR.png)
