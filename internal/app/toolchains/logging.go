package toolchains

import (
	"fmt"
	"os"
)

func emitInstallStep(format string, args ...any) {
	fmt.Fprintf(os.Stdout, format+"\n", args...)
}
