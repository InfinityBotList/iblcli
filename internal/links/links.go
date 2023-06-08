// Package links provides functions for getting certain base links of the IBL API
package links

import "os"

// Returns the frontend url
func GetFrontendURL() string {
	if os.Getenv("FRONTEND_URL") == "" {
		return "https://infinitybots.gg"
	} else {
		return os.Getenv("FRONTEND_URL")
	}
}

// Returns the cdn url
func GetCdnURL() string {
	if os.Getenv("CDN_URL") == "" {
		return "https://cdn.infinitybots.gg"
	} else {
		return os.Getenv("CDN_URL")
	}
}
