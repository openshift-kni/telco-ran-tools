package cmd

import (
	"strings"
	"testing"
)

func TestGenerateOcMirrorCommand(t *testing.T) {
	c := generateOcMirrorCommand("/tmp/fp-cli-lol")
	want := "-c /tmp/fp-cli-lol/imageset.yaml file:///tmp/fp-cli-lol/mirror --ignore-history --dry-run"

	if strings.Join(c.Args[1:], " ") != want {
		t.Errorf("got %s, want %s", c, want)
	}
}

func TestGenerateCreateArtifactsCommand(t *testing.T) {
	c := generateCreateArtifactsCommand("/tmp/fp-cli-lol")
	want := "/usr/bin/bash -c cat /tmp/fp-cli-lol/mirror/oc-mirror-workspace/mapping.txt | cut -d \"=\" -f1 > /tmp/fp-cli-lol/artifacts.txt"

	if c.String() != want {
		t.Errorf("got %s, want %s", c, want)
	}
}

func TestGenerateSkopeoCopyCommand(t *testing.T) {
	c := generateSkopeoCopyCommand("/tmp/mnt",
		"assisted-installer-agent-rhel8@sha256_54f7376e521a3b22ddeef63623fc7256addf62a9323fa004c7f48efa7388fe39",
		"registry.redhat.io/multicluster-engine/assisted-installer-agent-rhel8@sha256:54f7376e521a3b22ddeef63623fc7256addf62a9323fa004c7f48efa7388fe39")
	want := "copy docker://registry.redhat.io/multicluster-engine/assisted-installer-agent-rhel8@sha256:54f7376e521a3b22ddeef63623fc7256addf62a9323fa004c7f48efa7388fe39 dir:///tmp/mnt/assisted-installer-agent-rhel8@sha256_54f7376e521a3b22ddeef63623fc7256addf62a9323fa004c7f48efa7388fe39 -q"

	if strings.Join(c.Args[1:], " ") != want {
		t.Errorf("got %s, want %s", c, want)
	}
}

func TestTarArtifactCommand(t *testing.T) {
	c := generateTarArtifactCommand("/tmp/mnt", "assisted-installer-agent-rhel8@sha256_54f7376e521a3b22ddeef63623fc7256addf62a9323fa004c7f48efa7388fe39")
	want := "/usr/bin/tar czvf /tmp/mnt/assisted-installer-agent-rhel8@sha256_54f7376e521a3b22ddeef63623fc7256addf62a9323fa004c7f48efa7388fe39.tgz /tmp/mnt/assisted-installer-agent-rhel8@sha256_54f7376e521a3b22ddeef63623fc7256addf62a9323fa004c7f48efa7388fe39"

	if c.String() != want {
		t.Errorf("got %s, want %s", c, want)
	}
}

func TestGenerateRemoveArtifactCommand(t *testing.T) {
	c := generateRemoveArtifactCommand("/tmp/mnt", "assisted-installer-agent-rhel8@sha256_54f7376e521a3b22ddeef63623fc7256addf62a9323fa004c7f48efa7388fe39")
	want := "/usr/bin/rm -rf /tmp/mnt/assisted-installer-agent-rhel8@sha256_54f7376e521a3b22ddeef63623fc7256addf62a9323fa004c7f48efa7388fe39"

	if c.String() != want {
		t.Errorf("got %s, want %s", c, want)
	}
}
