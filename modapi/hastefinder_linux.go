//go:build freebsd || linux || netbsd

package modapi

func ResolveHastePath() (string, error) {
	linux_is_not_currently_supported_please_use_windows()

	return "", nil
}
