[![Documentation](https://godoc.org/github.com/phemmer/sawmill?status.png)](http://godoc.org/github.com/phemmer/sawmill)
[![Build status](https://travis-ci.org/phemmer/sawmill.svg?branch=master)](https://travis-ci.org/phemmer/sawmill)
[![Coverage status](https://coveralls.io/repos/phemmer/sawmill/badge.svg?branch=master)](https://coveralls.io/r/phemmer/sawmill?branch=master)

Sawmill is a flexible logging package for GO, with a strong emphasis on structured events.

By 'flexible', it is meant that sawmill has numerous configurable options out of the box. It is also designed for extensibility, being able to make use of third party packages.

The highlights:

* Can send events to multiple destinations in parallel.
* Supports writing to standard IO streams (STDOUT, files, etc), as well as external services (syslog, Splunk, Airbrake, etc).
* Default formatters automatically colorize output when sending to a terminal.
* Each destination can do it's own formatting, allowing you to do things like use one format for STDOUT, and another for syslog.
* Formats are fully customizable, using standard GO text templates.
* Serializes ancillary data, including nested structures (struct, map, slice, etc), into key/value format.
* Supports synchronous & asynchronous processing, allowing you to resume execution without waiting for external services to accept a message.
* ...and much more.

The project is fairly stable, but vendoring (e.g. [godep](https://github.com/tools/godep)) is recommended. Once [issue #29](https://github.com/phemmer/sawmill/issues/29) is resolved, I would consider the service as 1.0.

---
#### Example:

![Example output](http://i.imgur.com/3rfgVvk.png)

Source code:
```go
package main

import sm "github.com/phemmer/sawmill"
import "time"

type Message struct {
	Sender string
	Recipients []string
	Content string
}

func main() {
	defer sm.Stop()
	timeStart := time.Now()

	message := Message{"Me", []string{"World","Mars"}, "Hello!"}
	sm.Info("Message relayed", message)

	sm.Debug("Finished", sm.Fields{"duration": time.Now().Sub(timeStart)})
	sm.Fatal("Whoops!")
}
```


---

# Handlers

Handlers are the main workhorses of sawmill. After sawmill has generated an event, it routes it to all the registered handlers. Each handler can then process the event as it sees fit.

There are 2 main types of handlers, integration handlers, and utility handlers.  
Integration handlers send the event outside of the program. This might be as simple as writing to STDOUT, or something more advanced like Splunk.  
Utility handlers do something with the event internally. This could be filtering the events before sending them on to another handler, or storing the event in memory for later retrieval.

## Integration handlers

### [Airbrake](https://github.com/phemmer/sawmill/tree/master/handler/airbrake)

The airbrake handler sends events to the [Airbrake error reporting service](https://airbrake.io/).


Readme: https://github.com/phemmer/sawmill/blob/master/handler/airbrake/README.md  
Godoc: http://godoc.org/github.com/phemmer/sawmill/handler/airbrake

```go
import (
	"github.com/phemmer/sawmill"
	"github.com/phemmer/sawmill/handler/airbrake"
)

func main() {
	a := airbrake.New(123456, "0123456789abcdef0123456789abcdef", "production")
	filter := sawmill.FilterHandler(a).LevelMin(sawmill.ErrorLevel)
	sawmill.AddHandler("airbrake", filter)
}
```

![Example](http://i.imgur.com/jYIjk6s.png)

### [Sentry](https://github.com/phemmer/sawmill/tree/master/handler/sentry)

The sentry handler sends events to the [Sentry error reporting service](https://getsentry.com).


Readme: https://github.com/phemmer/sawmill/blob/master/handler/sentry/README.md  
Godoc: http://godoc.org/github.com/phemmer/sawmill/handler/sentry

```go
import (
	"github.com/phemmer/sawmill"
	"github.com/phemmer/sawmill/handler/sentry"
)

var sentryDSN = "https://00112233445566778899aabbccddeeff:0123456789abcdef0123456789abcdef@app.getsentry.com/12345"

func main() {
	if s, err := sentry.New(sentryDSN); err == nil {
		filter := sawmill.FilterHandler(s).LevelMin(sawmill.ErrorLevel)
		sawmill.AddHandler("sentry", filter)
	}
}
```

![Example](http://i.imgur.com/TsNFXgR.png)

### [Splunk](https://github.com/phemmer/sawmill/tree/master/handler/splunk)

The splunk handler sends events to a [Splunk log collector](http://www.splunk.com/). This includes both [Splunk Cloud](http://www.splunk.com/en_us/products/splunk-cloud.html) and [Splunk Enterprise](http://www.splunk.com/en_us/products/splunk-enterprise.html).

Readme: https://github.com/phemmer/sawmill/blob/master/handler/splunk/README.md  
Godoc: http://godoc.org/github.com/phemmer/sawmill/handler/splunk

```go
import (
	"github.com/phemmer/sawmill"
	"github.com/phemmer/sawmill/handler/splunk"
)

var splunkURL = "https://username:password@input-prd-p-xl29ahe4v1h3.cloud.splunk.com:8089/?index=development"

func main() {
	if s, err := splunk.New(splunkURL); err == nil {
		sawmill.AddHandler("splunk", s)
	}
}
```

![Example](http://i.imgur.com/ZRxSEte.png)

### [Syslog](https://github.com/phemmer/sawmill/tree/master/handler/syslog)

The syslog handler sends events to a syslog service. This can be a service running locally on the box, or remote.


Godoc: http://godoc.org/github.com/phemmer/sawmill/handler/syslog

### [Writer](https://github.com/phemmer/sawmill/tree/master/handler/writer)

The writer handler sends events to any `io.Writer` object. This can be STDOUT/STDERR, a normal file, or anything.  
The events can be formatted before being written out. The writer includes several pre-defined formats, including some which use colorization and tabulation to make the events easy to read on a console.

Readme: https://github.com/phemmer/sawmill/blob/master/handler/writer/README.md  
Godoc: http://godoc.org/github.com/phemmer/sawmill/handler/writer

## Utility handlers

### [Filter](https://github.com/phemmer/sawmill/tree/master/handler/filter)

The filter handler is used to filter events before sending them on to another handler. You can even chain multiple filter handlers together.

The most common use of this handler is to filter events based on their level, such as to send only error and above to an error reporting service.  
The handler can also perform basic deduplication, so that multiple identical events don't flood an integration handler.  
You can also provide your own function to determine whether to allow an event through.


Godoc: http://godoc.org/github.com/phemmer/sawmill/handler/filter
