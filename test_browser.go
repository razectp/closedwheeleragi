// Test program to verify browser functionality
package main

import (
	"fmt"
	"log"
	"time"

	"ClosedWheeler/pkg/browser"
)

func main() {
	fmt.Println("=== Browser Functionality Test ===")
	fmt.Println()

	// Create browser manager with visible mode
	opts := browser.DefaultOptions()
	opts.Headless = false // Force visible mode

	fmt.Printf("Creating browser manager (Headless: %v)...\n", opts.Headless)
	manager, err := browser.NewManager(opts)
	if err != nil {
		log.Fatalf("Failed to create browser manager: %v", err)
	}
	defer manager.Close()

	fmt.Println("✓ Browser manager created successfully")
	fmt.Println()

	// Test navigation
	taskID := "test-task"
	testURL := "https://example.com"

	fmt.Printf("Navigating to %s...\n", testURL)
	result, err := manager.Navigate(taskID, testURL)
	if err != nil {
		log.Fatalf("Navigation failed: %v", err)
	}

	fmt.Println("✓ Navigation successful!")
	fmt.Printf("  URL: %s\n", result.URL)
	fmt.Printf("  Title: %s\n", result.Title)
	fmt.Printf("  Status Code: %d\n", result.StatusCode)
	fmt.Printf("  Content length: %d characters\n", len(result.Content))
	fmt.Println()

	// Keep browser open for 5 seconds so user can see it
	fmt.Println("Browser will stay open for 5 seconds...")
	time.Sleep(5 * time.Second)

	// Get page text
	fmt.Println("Getting page text...")
	text, err := manager.GetPageText(taskID)
	if err != nil {
		log.Fatalf("Failed to get page text: %v", err)
	}
	fmt.Printf("✓ Page text retrieved (%d characters)\n", len(text))
	fmt.Println()

	// Close the tab
	fmt.Println("Closing browser tab...")
	if err := manager.ClosePage(taskID); err != nil {
		log.Fatalf("Failed to close page: %v", err)
	}
	fmt.Println("✓ Browser tab closed")
	fmt.Println()

	fmt.Println("=== All tests passed! ===")
}
