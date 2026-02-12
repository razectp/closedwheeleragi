package browser

import (
	"fmt"
	"log"

	"github.com/playwright-community/playwright-go"
)

// InstallDeps downloads the Playwright driver and Chromium browser.
// This is required before any browser automation can work.
// Works on Windows, macOS, and Linux automatically.
func InstallDeps() error {
	log.Println("[Browser] Installing Playwright driver and Chromium browser...")
	err := playwright.Install(&playwright.RunOptions{
		Browsers: []string{"chromium"},
		Verbose:  true,
	})
	if err != nil {
		return fmt.Errorf("failed to install Playwright browsers: %w", err)
	}
	log.Println("[Browser] Playwright installation complete.")
	return nil
}

// CheckDeps returns true if the Playwright driver is already installed and up-to-date.
func CheckDeps() bool {
	driver, err := playwright.NewDriver(&playwright.RunOptions{
		SkipInstallBrowsers: true,
		Verbose:             false,
	})
	if err != nil {
		return false
	}
	// Command("--version") will fail if the driver binary is missing
	cmd := driver.Command("--version")
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}
