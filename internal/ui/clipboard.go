package ui

import (
	"fmt"
	"os/exec"
	"strings"
)

// copyToClipboard pipes s to /usr/bin/pbcopy. We deliberately do not depend
// on github.com/atotto/clipboard (already an indirect dep via bubbles) so
// the data path stays explicit — pbcopy is part of the base macOS install
// and we are macOS-only anyway.
func copyToClipboard(s string) error {
	cmd := exec.Command("/usr/bin/pbcopy")
	cmd.Stdin = strings.NewReader(s)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pbcopy: %w", err)
	}
	return nil
}
