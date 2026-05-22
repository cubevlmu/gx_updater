package main

import (
	"fmt"

	"gx-update-server/internal/security"
)

func main() {
	cs, err := security.RandomBase64(32)
	if err != nil {
		fmt.Printf("Error generating client_secret: %v\n", err)
		return
	}

	pub, priv, err := security.GenerateEd25519KeyPair()
	if err != nil {
		fmt.Printf("Error generating Ed25519 keys: %v\n", err)
		return
	}

	fmt.Println("app_id: com.example.app")
	fmt.Println("name: Example App")
	fmt.Println("enabled: true")
	fmt.Printf("client_secret: \"%s\"\n", cs)
	fmt.Printf("ed25519_public_key: \"%s\"\n", pub)
	fmt.Printf("ed25519_private_key: \"%s\"\n", priv)
}
