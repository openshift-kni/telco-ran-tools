package cmd

const imageSetTemplate = `
---
apiVersion: mirror.openshift.io/v1alpha2
kind: ImageSetConfiguration
mirror:
  platform:
    channels:
      - name: stable-{{ .Release }}
  additionalImages:
    - name: quay.io/edge-infrastructure/assisted-installer-agent:latest
`
