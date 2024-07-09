# MSI

{% include-markdown "../includes/pro.md" comments=false %}

GoReleaser can create MSI installers for windows binaries using [msitools][].

The `msi` section specifies how the **installers** should be created:

```yaml
# .goreleaser.yaml
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
```

On Windows, it'll try to use the `candle` and `light` binaries from the
[Wix Toolkit][wix] instead.

Here's an example `wsx` file that you can build upon:

```xml
<?xml version='1.0' encoding='windows-1252'?>
<Wix xmlns='http://schemas.microsoft.com/wix/2006/wi'>
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
			Comments=''
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
			<Directory Id='ProgramFiles{{ if eq .Arch "amd64" }}64{{ end }}Folder' Name='PFiles'>
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
1. Only `amd64` and `386` are supported.

{% include-markdown "../includes/templates.md" comments=false %}

[msitools]: https://wiki.gnome.org/msitools
[wix]: https://wixtoolset.org
