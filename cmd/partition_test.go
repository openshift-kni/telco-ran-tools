package cmd

import (
	"testing"
)

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
