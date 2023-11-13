/*
Copyright 2023 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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
    - catalog: {{ .CatalogRedhatOperators }}
      packages:
        - name: multicluster-engine
          channels:
{{- /* Because there is no versionless "stable" channel, we need to include the latest versioned channel */ -}}
{{- if ne .MceChannel "2.4" }}
            - name: 'stable-2.4'
{{- end }}
            - name: 'stable-{{.MceChannel}}'
              minVersion: {{ .MceVersion }}
              maxVersion: {{ .MceVersion }}
{{- if .DuProfile }}
        - name: advanced-cluster-management
          channels:
  {{- /* Because there is no versionless "release" channel, we need to include the latest versioned channel */ -}}
  {{- if ne .AcmChannel "2.8" }}
            - name: 'release-2.8'
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
            - name: 'stable-5.8'
    - catalog: {{ .CatalogCertifiedOperators }}
      packages:
        - name: sriov-fec
          channels:
            - name: 'stable'
{{- end }}
`
