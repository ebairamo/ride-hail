package dto

import "fmt"

type Auth struct {
	Type  string `json:"type"`
	Token string `json:"token"`
}

func (a Auth) Validate() error {
	if a.Type != "auth" {
		return fmt.Errorf("invalid auth type")
	}
	if a.Token == "" {
		return fmt.Errorf("empty token")
	}
	return nil
}
