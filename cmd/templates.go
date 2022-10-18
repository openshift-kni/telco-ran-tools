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
{{- if .DuProfile }}
  operators:
    - catalog: registry.redhat.io/redhat/redhat-operator-index:v{{ .Channel }}
      packages:
#        - name: advanced-cluster-management
#          channels:
#             - name: 'release-2.6'
#             - name: 'release-2.5'
#        - name: multicluster-engine
#          channels:
#             - name: 'stable-2.1'
#             - name: 'stable-2.0'
        - name: local-storage-operator
          channels:
            - name: 'stable'
        - name: ptp-operator
          channels:
            - name: 'stable'
        - name: sriov-network-operator
          channels:
            - name: 'stable'
        - name: cluster-logging
          channels:
            - name: 'stable'
    - catalog: registry.redhat.io/redhat/certified-operator-index:v{{ .Channel }}
      packages:
        - name: sriov-fec
          channels:
            - name: 'stable'
{{- end }}
`
