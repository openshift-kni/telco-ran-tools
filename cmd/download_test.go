package cmd

import (
	"testing"
)

func TestGenerateOcMirrorCommand(t *testing.T) {
	c := generateOcMirrorCommand("/tmp/fp-cli-lol")
	want := "./oc-mirror -c imageset.yaml file:///tmp/fp-cli-lol/mirror --ignore-history --dry-run"

	if c.String() != want {
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
