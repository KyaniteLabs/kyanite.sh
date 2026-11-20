package cli

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// confirmYesNo prints a y/N confirmation prompt on out and reads the reply
// from in. Only "y" or "yes" (case-insensitive) confirms; empty input and
// every other reply cancel. This matches the `focus config reset` prompt.
func confirmYesNo(in io.Reader, out io.Writer, prompt string) bool {
	fmt.Fprintf(out, "%s (y/N): ", prompt)

	line, err := bufio.NewReader(in).ReadString('\n')
	if err != nil && line == "" {
		// No input at all (EOF): treat as empty, which cancels.
		fmt.Fprintln(out)
		return false
	}

	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes"
}

// readTypedYes reads one whitespace-delimited word from in and reports
// whether it is exactly "yes" (case-insensitive). Empty input, read errors
// and any other word cancel. This is the existing `focus bulk delete` gate.
func readTypedYes(in io.Reader) bool {
	var confirm string
	if _, err := fmt.Fscanln(in, &confirm); err != nil {
		return false
	}
	return strings.ToLower(confirm) == "yes"
}
