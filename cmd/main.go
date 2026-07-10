package main

import (
	"fmt"
)

const startupMessage = "auth service bootstrap is running"

func serviceMessage() string {
	return startupMessage
}

func main() {
	fmt.Println(serviceMessage())
}
