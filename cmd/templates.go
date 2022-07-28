package cmd

const imageSetTemplate = `
---
apiVersion: mirror.openshift.io/v1alpha2
kind: ImageSetConfiguration
mirror:
  platform:
    channels:
    - name: stable-{{ .Channel }}
      minVersion: {{ .Version }}
      maxVersion: {{ .Version }}
  additionalImages:
    - name: registry.redhat.io/multicluster-engine/assisted-installer-agent-rhel8
    - name: registry.redhat.io/multicluster-engine/assisted-installer-rhel8
`
