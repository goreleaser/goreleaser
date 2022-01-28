# Publish to Artifactory

## Publish to Artifactory using curl

Example of a [publishers](/customization/publishers/) section pushing files to a Artifactory instance using curl:

```yaml
publishers:
- name: artifactory
  cmd: >-
    curl -k -H "X-JFrog-Art-API: {{ .Env.ARTIFACTORY_API_KEY }}"
      -X PUT
      -H "Accept: application/json"
      -H "Content-Type: multipart/form-data"
      "https://artifactory.somehost.com/artifactory/my-repository/{{ tolower .Env.PROJECT_KEY }}/{{ tolower .ProjectName }}/{{ .Version }}/{{ .ArtifactName }}
      -T @{{ .ArtifactName }}
  dir: "{{ dir .ArtifactPath }}"
```

## Publish to Artifactory using jfrog cli

This assumes you have the [jfrog cli](https://jfrog.com/getcli/) downloaded and in your path, and configured with an API key

Example of a [publishers](/customization/publishers/) section pushing files to a Artifactory instance using jfrog cli with api key in config file:

```yaml
publishers:
- name: artifactory
  cmd: >-
    jfrog rt u "{{ .ArtifactName }}" "my-repository/{{ tolower .Env.PROJECT_KEY }}/{{ tolower .ProjectName }}/{{ .Version }}/"
  dir: "{{ dir .ArtifactPath }}"
```

Example of a [publishers](/customization/publishers/) section pushing files to a Artifactory instance using jfrog cli with api key in environment

```yaml
publishers:
- name: artifactory
  cmd: >-
    jfrog rt u "{{ .ArtifactName }}" "my-repository/{{ tolower .Env.PROJECT_KEY }}/{{ tolower .ProjectName }}/{{ .Version }}/" --api-key "{{ .Env.ARTIFACTORY_API_KEY }}"
  dir: "{{ dir .ArtifactPath }}"
```
