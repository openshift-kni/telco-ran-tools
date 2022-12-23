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
  {{- if ne $img "" }}
    - name: {{ $img }}
  {{- end }}
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
        - name: multicluster-engine
          channels:
{{- /* Because there is no versionless "stable" channel, we need to include the latest versioned channel */ -}}
{{- if ne .MceChannel "2.1" }}
            - name: 'stable-2.1'
{{- end }}
            - name: 'stable-{{.MceChannel}}'
              minVersion: {{ .MceVersion }}
              maxVersion: {{ .MceVersion }}
{{- if .DuProfile }}
        - name: advanced-cluster-management
          channels:
  {{- /* Because there is no versionless "release" channel, we need to include the latest versioned channel */ -}}
  {{- if ne .AcmChannel "2.6" }}
            - name: 'release-2.6'
  {{- end }}
            - name: 'release-{{ .AcmChannel }}'
              minVersion: {{ .AcmVersion }}
              maxVersion: {{ .AcmVersion }}
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
