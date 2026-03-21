package models

import "testing"

func TestEntry_Validate(t *testing.T) {
	tests := []struct {
		name    string
		entry   Entry
		wantErr string
	}{
		{
			name:    "valid entry",
			entry:   Entry{Key: "AAPL", Category: "watchlist", Value: "Apple Inc."},
			wantErr: "",
		},
		{
			name:    "empty key",
			entry:   Entry{Key: "", Category: "watchlist"},
			wantErr: "key is required",
		},
		{
			name:    "key too long",
			entry:   Entry{Key: string(make([]byte, 501)), Category: "test"},
			wantErr: "key must be 500 characters or less",
		},
		{
			name:    "category too long",
			entry:   Entry{Key: "test", Category: string(make([]byte, 201))},
			wantErr: "category must be 200 characters or less",
		},
		{
			name:    "value too long",
			entry:   Entry{Key: "test", Value: string(make([]byte, 100001))},
			wantErr: "value must be 100000 characters or less",
		},
		{
			name:    "empty category is valid",
			entry:   Entry{Key: "test"},
			wantErr: "",
		},
		{
			name:    "empty value is valid",
			entry:   Entry{Key: "test", Category: "cat"},
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.entry.Validate()
			if got != tt.wantErr {
				t.Errorf("Validate() = %q, want %q", got, tt.wantErr)
			}
		})
	}
}
