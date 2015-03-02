package sawmill_test

import (
	"github.com/phemmer/sawmill"
)

type mystruct struct {
	Foo  string
	Bar  string
	Baz  []byte
	List []int
}

func Example() {
	defer sawmill.Stop()

	data := &mystruct{
		Foo:  "FOO",
		Bar:  "BAR var",
		Baz:  []byte("abc\000def"),
		List: []int{4, 5, 6},
	}

	sawmill.Info("An event occurred", data)
	sawmill.Fatal("Whoops!", sawmill.Fields{"fu": "bar"})
}
