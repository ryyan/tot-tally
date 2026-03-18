/*
This script is used to manually reset the tot creation limit for a specific IP address.
The application limits each IP to a maximum of 10 tots to prevent spam. This limit
is tracked by storing a count in a file named after the SHA-256 hash of the IP
within the "limits/" directory.

Usage:

	go run scripts/reset_ip_limit.go <IP_ADDRESS>

Example:

	go run scripts/reset_ip_limit.go 192.168.1.1
*/
package main

import (
	"crypto/sha256"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run scripts/reset_ip_limit.go <IP_ADDRESS>")
	}

	ip := os.Args[1]
	// Match the hashing logic used in internal/storage/storage.go
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(ip)))
	path := filepath.Join("limits", hash)

	err := os.Remove(path)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("No limit record found for IP: %s (hash: %s)\n", ip, hash)
			return
		}
		log.Fatalf("Error removing limit file: %v", err)
	}

	fmt.Printf("Successfully reset limit for IP: %s (deleted file: %s)\n", ip, path)
}
