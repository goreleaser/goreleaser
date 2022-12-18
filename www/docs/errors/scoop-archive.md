# Scoop requires a windows archive

The Scoop pipe requires a Windows build and archive.

The archive should not be in `binary` format.

For instance, this won't work:

```yaml
archives:
- format: binary
```


But this would:

```yaml
archives:
- format: zip
```
