# Publish to Artifactory using jfrog cli

This cookbook is an example of a [publishers](../customization/publishers.md)
section that uses the [jfrog cli](https://jfrog.com/getcli/) to upload files to
Artifactory. It is an alternative to using the
[Artifactory Publisher](../customization/artifactory.md) to upload to
artifactory.

The benefit of this method is that it uses the jfrog cli configuration instead
of environment variables for configuration.

This assumes you have the [jfrog cli](https://jfrog.com/getcli/) downloaded and
in your path, and
[configured](https://www.jfrog.com/confluence/display/CLI/JFrog+CLI#JFrogCLI-JFrogPlatformConfiguration)
with an API key.

```yaml
publishers:
- name: artifactory
 cmd: >-
   jfrog rt u "{{ .ArtifactName }}" "my-repository/{{ tolower .Env.PROJECT_KEY }}/{{ tolower .ProjectName }}/{{ .Version }}/"
 dir: "{{ dir .ArtifactPath }}"
```

Example of a [publishers](../customization/publishers.md) section pushing files
to an Artifactory instance using jfrog cli with api key in environment

```yaml
publishers:
- name: artifactory
 cmd: >-
   jfrog rt u "{{ .ArtifactName }}" "my-repository/{{ tolower .Env.PROJECT_KEY }}/{{ tolower .ProjectName }}/{{ .Version }}/" --api-key "{{ .Env.ARTIFACTORY_API_KEY }}"
 dir: "{{ dir .ArtifactPath }}"
```
