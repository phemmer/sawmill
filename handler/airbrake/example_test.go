package airbrake_test

import (
	"os"
	"strings"

	"github.com/phemmer/sawmill"
	"github.com/phemmer/sawmill/handler/airbrake"
)

func Example() {
	sawmill.SetStackMinLevel(sawmill.ErrorLevel)

	a := airbrake.New(12345, "0123456789abcdef0123456789abcdef", "development")
	a.Context.URL = "http://myproject.example.com"
	// Add all environment variables.
	for _, envVar := range os.Environ() {
		envKP := strings.SplitN(envVar, "=", 2)
		a.Env[envKP[0]] = envKP[1]
	}
	filter := sawmill.FilterHandler(a).LevelMin(sawmill.ErrorLevel)
	sawmill.AddHandler("airbrake", filter)
}
