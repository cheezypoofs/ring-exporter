package exporter

import (
	"bufio"
	"fmt"
	_ "github.com/cheezypoofs/ring-exporter/ringapi"
	"golang.org/x/crypto/ssh/terminal"
	"os"
	"strings"
	"syscall"
)

type CliAuthenticator struct {
}

// PromptCredentials implements `ringapi.Authenticator` interface
func (*CliAuthenticator) PromptCredentials() (string, string, error) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter Username: ")
	username, _ := reader.ReadString('\n')

	fmt.Print("Enter Password: ")
	bytePassword, _ := terminal.ReadPassword(int(syscall.Stdin))
	password := string(bytePassword)
	fmt.Printf("\n")

	return strings.TrimSpace(username), strings.TrimSpace(password), nil
}

// Prompt2FACode implements `ringapi.Authenticator` interface
func (*CliAuthenticator) Prompt2FACode() (string, error) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter 2FA Code: ")
	code, _ := reader.ReadString('\n')
	return strings.TrimSpace(code), nil
}
