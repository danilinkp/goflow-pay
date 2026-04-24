package credentials

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	credsDirName  = ".goflow"
	credsFileName = "credentials"
)

type Credentials struct {
	Token     string `json:"token"`
	UserID    string `json:"user_id"`
	CompanyID string `json:"company_id"`
	Role      string `json:"role"`
	Email     string `json:"email"`
}

func credsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home dir: %w", err)
	}
	return filepath.Join(home, credsDirName, credsFileName), nil
}

func Save(creds *Credentials) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home dir: %w", err)
	}

	dir := filepath.Join(home, credsDirName)
	if err = os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create credentials dir: %w", err)
	}

	path := filepath.Join(dir, credsFileName)

	data, err := json.Marshal(creds)
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}

	if err = os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write credentials: %w", err)
	}

	return nil
}

func Load() (*Credentials, error) {
	path, err := credsPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("not logged in, run: goflow auth login")
		}
		return nil, fmt.Errorf("failed to read credentials: %w", err)
	}

	var creds Credentials
	if err = json.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("failed to parse credentials: %w", err)
	}

	return &creds, nil
}

func Delete() error {
	path, err := credsPath()
	if err != nil {
		return err
	}

	if err = os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete credentials: %w", err)
	}

	return nil
}

func Exists() bool {
	path, err := credsPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}
