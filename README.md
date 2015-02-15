***Note: Still under heavy development. Vendoring (E.G. [godep](https://github.com/tools/godep)) is highly recommended.***

Sawmill is a flexible logging package for GO, with a strong emphasis on structured events.

By 'flexible', it is meant that sawmill has numerous configurable options out of the box. It is also designed for extensibility, being able to make use of third party packages.

The highlights:

* Can send events to multiple destinations.
* Each destination is able to have it's own formatter, allowing you to do things like use one format for STDOUT, and another for syslog.
* Formats are fully customizable, using standard GO text templates.
* Supports writing to standard IO streams (STDOUT, STDERR, files, etc).
* Supports writing to syslog.
* Automatically colorizes output when sending to a terminal.
* Serializes ancillary data, including nested structures (struct, map, slice, etc), into key/value format.
* Asynchronous processing. Returns without waiting for the destination to accept the message.
* ...And much more.

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
