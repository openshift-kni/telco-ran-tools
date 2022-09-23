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
    - name: {{ .AIAgentImage }}
    - name: {{ .AIInstallerImage }}
{{- if ne .AIControllerImage "" }}
    - name: {{ .AIControllerImage }}
{{- end }}
{{- range $img := .AdditionalImages }}
    - name: {{ $img }}
{{- end }}
#
# Example operators specification:
#
#  operators:
#    - catalog: registry.redhat.io/redhat/redhat-operator-index:v{{ .Channel }}
#      full: true
#      packages:
#        - name: ptp-operator
#          channels:
#            - name: 'stable'
#        - name: sriov-network-operator
#          channels:
#            - name: 'stable'
#        - name: cluster-logging
#          channels:
#            - name: 'stable'
{{- if eq .Channel "4.10" }}
  operators:
    - catalog: registry.redhat.io/redhat/redhat-operator-index:v{{ .Channel }}
      full: true
      packages:
        - name: odf-lvm-operator
          channels:
            - name: 'stable-{{ .Channel }}'
        - name: performance-addon-operator
          channels:
            - name: '{{ .Channel }}'
        - name: ptp-operator
          channels:
            - name: 'stable'
        - name: sriov-network-operator
          channels:
            - name: 'stable'
        - name: cluster-logging
          channels:
            - name: 'stable'
        - name: ocs-operator
          channels:
            - name: 'stable-{{ .Channel }}'
        - name: local-storage-operator
          channels:
            - name: 'stable'
    - catalog: registry.redhat.io/redhat/certified-operator-index:v{{ .Channel }}
      full: true
      packages:
        - name: sriov-fec
          channels:
            - name: 'stable'
{{- end }}
`
