package handler

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type TestStruct struct {
	Platform string `validate:"platform"`
	Username string `validate:"required,max=100,excludesall=\x00\n\r\t"`
	Quantity int    `validate:"min=1,max=10000"`
}

func TestValidator(t *testing.T) {
	// Ensure validator is initialized
	InitValidator()
	v := GetValidator()

	tests := []struct {
		name    string
		input   TestStruct
		wantErr bool
	}{
		{
			name: "Valid input",
			input: TestStruct{
				Platform: "twitch",
				Username: "validUser",
				Quantity: 10,
			},
			wantErr: false,
		},
		{
			name: "Invalid platform",
			input: TestStruct{
				Platform: "invalid",
				Username: "validUser",
				Quantity: 10,
			},
			wantErr: true,
		},
		{
			name: "Empty platform (allowed by platform validator if not required)",
			input: TestStruct{
				Platform: "",
				Username: "validUser",
				Quantity: 10,
			},
			wantErr: false,
		},
		{
			name: "Invalid username (too long)",
			input: TestStruct{
				Platform: "twitch",
				Username: string(make([]byte, 101)),
				Quantity: 10,
			},
			wantErr: true,
		},
		{
			name: "Invalid username (control chars)",
			input: TestStruct{
				Platform: "twitch",
				Username: "user\nname",
				Quantity: 10,
			},
			wantErr: true,
		},
		{
			name: "Invalid quantity (too low)",
			input: TestStruct{
				Platform: "twitch",
				Username: "validUser",
				Quantity: 0,
			},
			wantErr: true,
		},
		{
			name: "Invalid quantity (too high)",
			input: TestStruct{
				Platform: "twitch",
				Username: "validUser",
				Quantity: 10001,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateStruct(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
