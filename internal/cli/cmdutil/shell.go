package cmdutil

import (
	"os/exec"
	"strconv"
	"strings"
)

// DisplayShellCommand formats an exec.Cmd as a human-readable shell command,
// quoting arguments that contain whitespace or double-quote characters.
func DisplayShellCommand(cmd *exec.Cmd) string {
	items := make([]string, 0, len(cmd.Args))
	for _, arg := range cmd.Args {
		if strings.ContainsAny(arg, " \t\n\"") {
			items = append(items, strconv.Quote(arg))
			continue
		}
		items = append(items, arg)
	}
	return strings.Join(items, " ")
}
