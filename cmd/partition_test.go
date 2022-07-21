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

func TestGenerateGetDeviceSizeCommand(t *testing.T) {
	c := generateGetDeviceSizeCommand("/dev/lol")
	want := "/usr/bin/lsblk /dev/lol -osize -dn"

	if c.String() != want {
		t.Errorf("got %s, want %s", c, want)
	}
}

func TestGeneratePartitionCommand(t *testing.T) {
	c := generatePartitionCommand("/dev/lol", 100)
	want := "/usr/sbin/sgdisk -n 1:-100GiB:0 /dev/lol -g -c:1:data"

	if c.String() != want {
		t.Errorf("got %s, want %s", c, want)
	}
}

func TestGenerateFormatCommand(t *testing.T) {
	c := generateFormatCommand("/dev/lol")
	want := "/usr/sbin/mkfs.xfs -f /dev/lol1"

	if c.String() != want {
		t.Errorf("got %s, want %s", c, want)
	}
}

func TestGenerateMountCommand(t *testing.T) {
	c := generateMountCommand("/dev/lol", "/lol/mnt")
	want := "/usr/bin/mount /dev/lol1 /lol/mnt"

	if c.String() != want {
		t.Errorf("got %s, want %s", c, want)
	}
}
