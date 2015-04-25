The airbrake handler provides a handler for sawmill which sends events to the [airbrake](http://airbrake.io) service.

Example usage:

```go
package main

import (
	"os"
	"strings"

	"github.com/phemmer/sawmill"
	"github.com/phemmer/sawmill/handler/airbrake"
)

func main() {
	defer sawmill.Stop()
	sawmill.SetStackMinLevel(sawmill.ErrorLevel)

	a := airbrake.New(123456, "0123456789abcdef0123456789abcdef", "production")
	a.Context.URL = "http://myproject.example.com"
	// Add all environment variables.
	for _, envVar := range os.Environ() {
		envKP := strings.SplitN(envVar, "=", 2)
		a.Env[envKP[0]] = envKP[1]
	}
	filter := sawmill.FilterHandler(a).LevelMin(sawmill.ErrorLevel)
	sawmill.AddHandler("airbrake", filter)

	_, err := os.Create("/test")
	if err != nil {
		sawmill.Error("Failed to create /test", sawmill.Fields{"error": err, "path": "/test"})
	}
}
```

---

### General
![General](http://i.imgur.com/jYIjk6s.png)

---
### Backtrace
![Backtrace](http://i.imgur.com/pDbJ9ok.png)

---
### Params
![Params](http://i.imgur.com/pW2KgQh.png)

---
### Env
![Env](http://i.imgur.com/IhpVN8r.png)

---
### Context
![Context](http://i.imgur.com/CughP6P.png)
