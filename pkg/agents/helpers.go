package agents

import (
	"fmt"
	"os/exec"
)

// runShellCmd выполняет shell-команду
func runShellCmd(cmd string) error {
	out, err := exec.Command("sh", "-c", cmd).CombinedOutput()
	if err != nil {
		return fmt.Errorf("shell команда: %w\n%s", err, string(out))
	}
	return nil
}
