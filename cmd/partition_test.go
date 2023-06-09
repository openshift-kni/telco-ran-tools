package cmd

import (
	"testing"
)

func TestGeneratePartitionCommand(t *testing.T) {
	c := generatePartitionCommand("/dev/lol", "data", 1, 100)
	want := "/usr/sbin/sgdisk -n 1:-100GiB:0 /dev/lol -g -c:1:data"

	if c.String() != want {
		t.Errorf("got %s, want %s", c, want)
	}
}

func TestGeneratePartitionLabelCommand(t *testing.T) {
	c := generatePartitionCommand("/dev/nvme2", "factory-prestaging", 5, 250)
	want := "/usr/sbin/sgdisk -n 5:-250GiB:0 /dev/nvme2 -g -c:5:factory-prestaging"

	if c.String() != want {
		t.Errorf("got %s, want %s", c, want)
	}
}

func TestGenerateFormatCommand(t *testing.T) {
	c := generateFormatCommand("/dev/lol", 1)
	want := "/usr/sbin/mkfs.xfs -f /dev/lol1"

	if c.String() != want {
		t.Errorf("got %s, want %s", c, want)
	}
}

func TestGenerateFormatNvmeCommand(t *testing.T) {
	c := generateFormatCommand("/dev/nvme2", 5)
	want := "/usr/sbin/mkfs.xfs -f /dev/nvme2p5"

	if c.String() != want {
		t.Errorf("got %s, want %s", c, want)
	}
}
