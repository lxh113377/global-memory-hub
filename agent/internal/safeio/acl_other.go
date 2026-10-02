//go:build !windows

package safeio

// HardenTokenACL 非 Windows 平台为 no-op (POSIX filemode 0o600 已生效)。
func HardenTokenACL(path string) error { return nil }
