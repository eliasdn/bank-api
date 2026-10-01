package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUser_SetPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{"valid password", "StrongPass123!", false},
		{"too short", "Pass1!", true},
		{"no uppercase", "password123!", true},
		{"no lowercase", "PASSWORD123!", true},
		{"no number", "Password!", true},
		{"no special char", "Password123", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &User{}
			err := u.SetPassword(tt.password)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, u.PasswordHash)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, u.PasswordHash)
				assert.True(t, u.CheckPassword(tt.password))
			}
		})
	}
}
