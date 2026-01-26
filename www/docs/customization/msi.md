# MSI

<!-- md:pro -->

GoReleaser can create MSI installers for windows binaries using [msitools][].

The `msi` section specifies how the **installers** should be created:

```yaml title=".goreleaser.yaml"
msi:
  - # ID of the resulting installer.
    #
    # Default: the project name.
    id: foo

    # Filename of the installer (without the extension).
    #
    # Default: '{{.ProjectName}}_{{.MsiArch}}'.
    # Templates: allowed.
    name: "myproject-{{.MsiArch}}"

    # The WXS file used to create the installers.
    # The file contents go through the templating engine, so you can do things
    # like `{{.Version}}` inside of it.
    #
    # Templates: allowed.
    # Required.
    wxs: ./windows/app.wsx

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

    # More files that will be available in the context in which the installer
    # will be built.
    extra_files:
      - logo.ico

    # Sets extensions to msitools/wix.
    # See: https://wixtoolset.org/docs/v3/howtos/general/extension_usage_introduction/
    #
    # Templates: allowed.
    # <!-- md:inline_version v2.6 -->.
    extensions:
      - '{{ if eq .Runtime.Goos "windows" }}WixUIExtension{{ end }}'
      - "WixUtilExtension"

    # Whether to disable this particular MSI configuration.
    #
    # Templates: allowed.
    # <!-- md:inline_version v2.12 -->.
    disable: "{{ .IsSnapshot }}"

    # Whether to remove the archives from the artifact list.
    # If left as false, your end release will have both the zip and the msi
    # files.
    replace: true

    # Set the modified timestamp on the output installer, typically
    # you would do this to ensure a build was reproducible.
    # Pass an empty string to skip modifying the output.
    #
    # Templates: allowed.
    mod_timestamp: "{{ .CommitTimestamp }}"

    # Schema version to use.
    # msitools only supports v3.
    # wixtoolset v3 supports v3.
    # wix v4/v5 supports v4.
    #
    # Valid options: 'v3', 'v4'.
    # Default: inferred from the .wxs file.
    # <!-- md:inline_version v2.7 -->.
    version: v4

    # Before and after hooks for each MSI.
    # This feature is only available in GoReleaser Pro.
    # <!-- md:inline_version v2.14-unreleased -->.
    #
    # The after hooks have access to the MSI artifact, so you can use:
    # - {{ .ArtifactPath }} - full path to the MSI file
    # - {{ .ArtifactName }} - filename (e.g., foo_x64.msi)
    # - {{ .ArtifactExt }} - extension (.msi)
    hooks:
      before:
        - make clean # simple string
        - cmd: go generate ./... # specify cmd
        - cmd: go mod tidy
          output: true # always prints command output
          dir: ./submodule # specify command working directory
        - cmd: touch {{ .Env.FILE_TO_TOUCH }}
          env:
            - "FILE_TO_TOUCH=something-{{ .ProjectName }}" # specify hook level environment variables

      after:
        - cmd: codesign {{ .ArtifactPath }} # sign the MSI
        - cmd: cat *.yaml
          dir: ./submodule
        - cmd: touch {{ .Env.RELEASE_DONE }}
          env:
            - "RELEASE_DONE=something-{{ .ProjectName }}" # specify hook level environment variables
```

On Windows, it'll try to use the `candle` and `light` binaries from the
[Wix Toolkit][wix] instead if schema is v3. It'll use `wix` otherwise..

If you use any extensions, make sure to install them first. You can do so with
`wix extension add -g <extension name>`.

Here's an example `wsx` file that you can build upon:

=== "Schema v4"

    ```xml
    <Wix xmlns="http://wixtoolset.org/schemas/v4/wxs">
      <Package
        Name="{{.ProjectName}} {{.Version}}"
        UpgradeCode="ABCDDCBA-7349-453F-94F6-BCB5110BA4FD"
        Language="1033"
        Codepage="1252"
        Version="{{.Version}}"
        Manufacturer="My Company"
        InstallerVersion="200"
        ProductCode="ABCDDCBA-86C7-4D14-AEC0-86416A69ABDE">
        <SummaryInformation
          Keywords="Installer"
          Description="{{.ProjectName}} installer"
          Manufacturer="My Company" />
        <Media
          Id="1"
          Cabinet="Sample.cab"
          EmbedCab="yes"
          DiskPrompt="CD-ROM #1" />
        <Property
          Id="DiskPrompt"
          Value="{{.ProjectName}} {{.Version}} Installation [1]" />
        <Feature
          Id="Complete"
          Level="1">
          <ComponentRef Id="MainExecutable" />
        </Feature>
        <StandardDirectory Id='ProgramFiles{{ if eq .Arch "amd64" }}64{{ end }}Folder'>
          <Directory
            Id="{{.ProjectName}}"
            Name="{{.ProjectName}}">
            <Component
              Id="MainExecutable"
              Guid="ABCDDCBA-83F1-4F22-985B-FDB3C8ABD471">
              <File
                Id="{{.Binary}}exe"
                Name="{{.Binary}}.exe"
                DiskId="1"
                Source="{{.Binary}}.exe"
                KeyPath="yes" />
            </Component>
          </Directory>
        </StandardDirectory>
      </Package>
    </Wix>
    ```

=== "Schema v3"

    ```xml
    <?xml version='1.0' encoding='windows-1252'?>
    <Wix xmlns='http://schemas.microsoft.com/wix/2006/wi'>
      {{ if eq .MsiArch "x64" }}
      <?define ArchString = "(64 bit)" ?>
      <?define Win64 = "yes" ?>
      <?define ProgramFilesFolder = "ProgramFiles64Folder" ?>
      {{ else }}
      <?define ArchString = "" ?>
      <?define Win64 = "no" ?>
      <?define ProgramFilesFolder = "ProgramFilesFolder" ?>
      {{ end }}
      <Product
        Name='{{.ProjectName}} {{.Version}}'
        Id='ABCDDCBA-86C7-4D14-AEC0-86413A69ABDE'
        UpgradeCode='ABCDDCBA-7349-453F-94F6-BCB5110BA8FD'
        Language='1033'
        Codepage='1252'
        Version='{{.Version}}'
        Manufacturer='My Company'>

        <Package
          Id='*'
          Keywords='Installer'
          Description="{{.ProjectName}} installer"
          Manufacturer='My Company'
          InstallerVersion='200'
          Languages='1033'
          Compressed='yes'
          SummaryCodepage='1252'
        />

        <Media
          Id='1'
          Cabinet='Sample.cab'
          EmbedCab='yes'
          DiskPrompt="CD-ROM #1"
        />

        <Property
          Id='DiskPrompt'
          Value="{{.ProjectName}} {{.Version}} Installation [1]"
        />

        <Directory Id='TARGETDIR' Name='SourceDir'>
          <Directory Id='ProgramFilesFolder' Name='PFiles'>
            <Directory Id='{{.ProjectName}}' Name='{{.ProjectName}}'>
              <Component
                Id='MainExecutable'
                Guid='ABCDDCBA-83F1-4F22-985B-FDB3C8ABD474'
              >
                <File
                  Id='{{.Binary}}.exe'
                  Name='{{.Binary}}.exe'
                  DiskId='1'
                  Source='{{.Binary}}.exe'
                  KeyPath='yes'
                />
              </Component>
            </Directory>
          </Directory>
        </Directory>

        <Feature Id='Complete' Level='1'>
          <ComponentRef Id='MainExecutable' />
        </Feature>
      </Product>
    </Wix>
    ```

## Limitations

1. Some options available in the [Wix Toolset][wix] won't work with
   [msitools][], run a snapshot build and verify the generated installers.
   Also note that [msitools][] only supports some parts of the v3 schema.
1. Only `amd64` and `386` are supported.
   `arm64` support was added in GoReleaser v2.7.
1. Be mindful of schema versions. Also worth noting that extension names might
   be different in v4[^exts].

<!-- md:templates -->

[msitools]: https://wiki.gnome.org/msitools
[wix]: https://wixtoolset.org

[^exts]: See [documentation](https://wixtoolset.org/docs/fourthree/faqs/#wixext34) for reference.
