# Source RPMs

!!! warning

    Source RPM generation is in alpha and currently requires that you write
    your own `.spec` file template.

GoReleaser can generate Source RPM (`.src.rpm`) packages.

You will need to create `project.spec.tmpl`, which is a template for the [RPM
`.spec` file](https://rpm-software-management.github.io/rpm/manual/spec.html).

Available options:

```yaml title=".goreleaser.yaml"
source:
  # you must enable source archives to enable srpms
  enabled: true
  prefix_template: '{{ .ProjectName }}-{{ .Version }}/'

srpms:
  enabled: true
  spec_template_file: project.spec.tmpl
```
