package main

import (
	"fmt"

	"github.com/AlexanderGrooff/ansible-jinja-go/pkg/ansiblejinja"
)

func main() {
	fmt.Println("Ansible Jinja Go")
	// Example usage (will be implemented later)
	templatedString, err := ansiblejinja.TemplateString("Hello {{ name }}", map[string]interface{}{"name": "World"})
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Println(templatedString)
}
