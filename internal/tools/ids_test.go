package tools

import "testing"

func TestValidID(t *testing.T) {
	const id = "cdl0ajd5jtbs73abcdef"
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{name: "valid", in: id, want: true},
		{name: "empty", in: "", want: false},
		{name: "19 chars", in: id[:19], want: false},
		{name: "21 chars", in: id + "a", want: false},
		{name: "uppercase", in: "CDL0AJD5JTBS73ABCDEF", want: false},
		{name: "contains w", in: "cdl0ajd5jtbs73abcdew", want: false},
		{name: "path traversal", in: "../x", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := validID(tt.in); got != tt.want {
				t.Errorf("validID(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}
