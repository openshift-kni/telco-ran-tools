package cmd

import (
	"fmt"
	"testing"
)

func TestIsPartitionSizeTooBig(t *testing.T) {
	var tests = []struct {
		a, b float64
		want bool
	}{
		{float64(120), float64(100), false},
		{float64(120), float64(120), false},
		{float64(120), float64(121), true},
	}

	for _, tt := range tests {
		testname := fmt.Sprintf("%f,%f", tt.a, tt.b)
		t.Run(testname, func(t *testing.T) {
			ans := isPartitionSizeTooBig(tt.a, tt.b)
			if ans != tt.want {
				t.Errorf("got %t, want %t", ans, tt.want)
			}
		})
	}
}

func TestGenerateGetDeviceSizeCOmmand(t *testing.T) {
	c := generateGetDeviceSizeCommand("/dev/lol")
	want := "/usr/bin/lsblk /dev/lol -osize -dn"

	if c.String() != want {
		t.Errorf("got %s, want %s", c, want)
	}
}
