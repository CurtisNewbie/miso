package cli

import "os/exec"

// Get dependency package version in current working directory.
func PkgVersion(location string, pkg string) (string, error) {
	o, err := Run("go", []string{"list", "-m", "-f", `{{ .Version }}`, pkg}, func(c *exec.Cmd) {
		c.Dir = location
	})
	return string(o), err
}
