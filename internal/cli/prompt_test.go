package cli

import "testing"

func TestParseConfirmation(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    bool
		wantErr bool
	}{
		{name: "yes short", input: "y\n", want: true},
		{name: "yes long", input: "yes", want: true},
		{name: "no short", input: "n", want: false},
		{name: "no long", input: "no\n", want: false},
		{name: "invalid", input: "maybe", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseConfirmation(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}

				return
			}

			if err != nil {
				t.Fatalf("parseConfirmation returned error: %v", err)
			}

			if got != tt.want {
				t.Fatalf("unexpected confirmation: got %v want %v", got, tt.want)
			}
		})
	}
}
