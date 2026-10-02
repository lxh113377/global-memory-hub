//go:build windows

package safeio

import (
	"fmt"
	"os"
	"os/exec"
)

// HardenTokenACL 把文件 ACL 收紧到当前用户 (NTFS 上 POSIX filemode 0o600 不生效,
// G5: 显式去掉继承并只授予本用户)。
func HardenTokenACL(path string) error {
	user := os.Getenv("USERNAME")
	if user == "" {
		return fmt.Errorf("USERNAME env empty; cannot set ACL")
	}
	out, err := exec.Command("icacls", path, "/inheritance:r", "/grant:r", user+":F").CombinedOutput()
	if err != nil {
		return fmt.Errorf("icacls: %v: %s", err, string(out))
	}
	return nil
}
