// Package version содержит информацию о версии и дате сборки бинарника,
// заполняемую на этапе компиляции через ldflags.
package version

import "fmt"

var (
	Version   = "dev"
	BuildDate = "unknown"
)

func String(component string) string {
	return fmt.Sprintf("GophKeeper %s %s (built %s)", component, Version, BuildDate)
}
