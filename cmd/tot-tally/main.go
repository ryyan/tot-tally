// main.go is the entry point for the application. It starts the web server and background tasks.
package main

import totWeb "tot-tally/internal/web"

func main() {
	// Start the web server and all associated background tasks.
	// All configuration and initialization logic is encapsulated within the web package.
	totWeb.Start()
}
