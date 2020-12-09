# Release a library

Maybe you don't want to actually release binaries, but just generate a changelog and whatnot for your Go libraries? GoReleaser got you covered!

All you need is to add `skip: true` to the build config:

```yaml
builds:
- skip: true
```
