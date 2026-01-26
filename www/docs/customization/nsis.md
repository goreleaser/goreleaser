# NSIS

<!-- md:pro -->
<!-- md:version v2.14-unreleased -->

GoReleaser can create Nullsoft Scriptable Install System (NSIS) installers for
Windows binaries using [makensis][].

The `nsis` section specifies how the installers should be created:

```yaml title=".goreleaser.yaml"
nsis:
  - # ID of the resulting installer.
    #
    # Default: the project name.
    id: foo

    # Filename of the installer (without the extension).
    #
    # Your NSIS script should use `{{.Name}}.exe` in the `OutFile` directive
    # to match this name.
    #
    # Default: '{{.ProjectName}}_{{.Arch}}_setup'.
    # Templates: allowed.
    name: "myproject-{{.Arch}}"

    # The NSIS script file used to create the installers.
    # The file contents go through the templating engine, so you can do things
    # like `{{.Version}}` and `{{.Name}}` inside of it.
    #
    # **Important**: Your script must include `OutFile "{{.Name}}.exe"`
    # for the installer to be output with the correct file name.
    #
    # Templates: allowed.
    # Required.
    script: ./windows/installer.nsi

    # IDs of the archives to use.
    # Empty means all IDs.
    ids:
      - foo
      - bar

    # GOAMD64 to specify which amd64 version to use if there are multiple
    # versions from the build section.
    #
    # Default: v1.
    goamd64: v1

    # Extra files to include in the installer build context.
    # You can use glob patterns to include multiple files:
    extra_files:
      - glob: "README*.md"
        name_template: "{{ .ProjectName }}_README.md"

    # Additional templated extra files to add to the installer.
    # Those files will have their contents pass through the template engine,
    # and its results will be added to the package.
    templated_extra_files:
      - src: "LICENSE_TEMPLATE.md"
        dst: "LICENSE_{{.Version}}.txt"
      - src: "config/config.yaml.templ"
        dst: "config/config.yaml"

    # Whether to disable this particular NSIS configuration.
    #
    # Templates: allowed.
    # <!-- md:inline_version v2.12 -->.
    disable: "{{ .IsSnapshot }}"

    # Whether to remove the archives from the artifact list.
    # If left as false, your end release will have both the zip and the exe
    # files.
    replace: true

    # Set the modified timestamp on the output installer, typically
    # you would do this to ensure a build was reproducible.
    # Pass an empty string to skip modifying the output.
    #
    # Templates: allowed.
    mod_timestamp: "{{ .CommitTimestamp }}"
```

## Template Variables

The following template variables are available for use in your NSIS scripts
(in addition to standard GoReleaser variables):

- `.Name` - Name as defined in `nsis.name`
- `.Arch` - NSIS architecture: `x86` (386), `x64` (amd64), or `arm64`
- `.ProgramFiles` - Architecture-specific Program Files path:
  `$PROGRAMFILES64` (amd64) or `$PROGRAMFILES` (others)

## Example NSIS Script

```nsi title="installer.nsi"
!define APP_NAME "{{ .ProjectName }}"
!define APP_VERSION "{{ .Version }}"

OutFile "{{ .Name }}.exe"
!include "MUI2.nsh"

Name "${APP_NAME} ${APP_VERSION}"
InstallDir "{{ .ProgramFiles }}\${APP_NAME}"
RequestExecutionLevel admin

!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_LICENSE "LICENSE"
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH
!insertmacro MUI_LANGUAGE "English"

Section "Main Section" SEC01
  SetOutPath "$INSTDIR"
  File "{{ .Binary }}"
  CreateDirectory "$SMPROGRAMS\${APP_NAME}"
  CreateShortcut "$SMPROGRAMS\${APP_NAME}\${APP_NAME}.lnk" "$INSTDIR\{{ .Binary }}"
  WriteUninstaller "$INSTDIR\uninstall.exe"
SectionEnd

Section "Uninstall"
  Delete "$INSTDIR\{{ .Binary }}"
  Delete "$INSTDIR\uninstall.exe"
  RMDir "$INSTDIR"
  Delete "$SMPROGRAMS\${APP_NAME}\${APP_NAME}.lnk"
  RMDir "$SMPROGRAMS\${APP_NAME}"
SectionEnd
```

<!-- md:templates -->

[makensis]: https://nsis.sourceforge.io/
