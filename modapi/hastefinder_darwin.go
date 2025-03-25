//go:build darwin

package modapi

func ResolveHastePath() (string, error) {
	mac_is_not_currently_supported_please_use_windows()

	return "", nil
}
