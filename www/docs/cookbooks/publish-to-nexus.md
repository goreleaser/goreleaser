# Publish to Nexus

Example of a [publishers](/customization/publishers/) section pushing files to a Nexus instance:

```yaml
publishers:
- name: nexus
  cmd: >-
    curl -k -u "{{ .Env.NEXUS_USERNAME }}:{{ .Env.NEXUS_PASSWORD }}"
      -X POST
      -H "Accept: application/json"
      -H "Content-Type: multipart/form-data"
      "https://nexuspro.somehost.com/service/rest/v1/components?repository=go-raw-autopub"
      -F "raw.directory={{ tolower .Env.PROJECT_KEY }}/{{ tolower .ProjectName }}/{{ .Version }}"
      -F "raw.asset1=@{{ .ArtifactName }};type=application/gzip"
      -F "raw.asset1.filename={{ .ArtifactName }}"
  dir: "{{ dir .ArtifactPath }}"
```
