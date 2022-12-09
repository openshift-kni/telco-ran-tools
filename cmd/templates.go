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
#    - catalog: registry.redhat.io/redhat/redhat-operator-index:v4.11
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
  operators:
    - catalog: registry.redhat.io/redhat/redhat-operator-index:v{{ .Channel }}
      packages:
        - name: advanced-cluster-management
          channels:
             - name: 'release-2.6'
  {{- if eq (slice .HubVersion 0 4) "2.6."}}
               minVersion: {{ .HubVersion }}
               maxVersion: {{ .HubVersion }}
  {{- end }}
  {{- if eq (slice .HubVersion 0 4) "2.5."}}
             - name: 'release-2.5'
               minVersion: {{ .HubVersion }}
               maxVersion: {{ .HubVersion }}
  {{- end }}
        - name: multicluster-engine
          channels:
             - name: 'stable-2.1'
  {{- if eq (slice .HubVersion 0 4) "2.6."}}
               minVersion: 2.1.{{ slice .HubVersion 4 5 }}
               maxVersion: 2.1.{{ slice .HubVersion 4 5 }}
  {{- end }}
  {{- if eq (slice .HubVersion 0 4) "2.5."}}
             - name: 'stable-2.0'
               minVersion: 2.0.{{ slice .HubVersion 4 5 }}
               maxVersion: 2.0.{{ slice .HubVersion 4 5 }}
  {{- end}}        
  {{- if .DuProfile }}
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
  {{- if eq .Channel "4.12" }}
        - name: odf-lvm-operator
        channels:
          - name: 'stable-4.12'
        - name: amq7-interconnect-operator
          channels:
            - name: '1.10.x'
        - name: bare-metal-event-relay
          channels:
            - name: 'stable'
  {{- end }}
    - catalog: registry.redhat.io/redhat/certified-operator-index:v{{ .Channel }}
      packages:
        - name: sriov-fec
          channels:
            - name: 'stable'
{{- end }}
`
