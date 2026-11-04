// Command unityfake is a tiny stand-in for the Unity editor used only by
// tooling/unity-test.wrapper.test.ps1 to exercise the real wrapper code paths.
//
// It accepts the same arguments the wrapper passes (ignoring the rest), writes
// whatever NUnit XML the test asks for (or nothing at all) and exits with a
// configurable code. It is never used by CI, developers or the editor itself.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func main() {
	testResults := argument("testResults")
	logFile := argument("logFile")

	// Emit whatever the test wants on stdout/stderr, simulating raw editor output.
	if noise := os.Getenv("TTG_FAKE_STDOUT"); noise != "" {
		fmt.Println(noise)
	}
	if noise := os.Getenv("TTG_FAKE_STDERR"); noise != "" {
		fmt.Fprintln(os.Stderr, noise)
	}

	if logFile != "" {
		_ = os.MkdirAll(filepath.Dir(logFile), 0o755)
		_ = os.WriteFile(logFile, []byte("unityfake: batch mode stand-in\n"), 0o644)
	}

	if testResults != "" {
		if content := os.Getenv("TTG_FAKE_XML"); content != "" {
			_ = os.MkdirAll(filepath.Dir(testResults), 0o755)
			if err := os.WriteFile(testResults, []byte(content), 0o644); err != nil {
				os.Exit(90)
			}
		}
	}

	exitCode, err := strconv.Atoi(os.Getenv("TTG_FAKE_EXIT"))
	if err != nil {
		exitCode = 0
	}
	os.Exit(exitCode)
}

// argument returns the value of -name value or -name=value, ignoring unknown
// arguments so the fake tolerates the full editor command line.
func argument(name string) string {
	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		current := args[i]
		if current == "-"+name || current == "--"+name {
			if i+1 < len(args) {
				return args[i+1]
			}
			return ""
		}
		if strings.HasPrefix(current, "-"+name+"=") {
			return strings.TrimPrefix(current, "-"+name+"=")
		}
		if strings.HasPrefix(current, "--"+name+"=") {
			return strings.TrimPrefix(current, "--"+name+"=")
		}
	}
	return ""
}
