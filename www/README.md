# Documentation

Documentation is written in mkdocs and there are a few extensions that allow richer
authoring than markdown.

To iterate with documentation, therefore, it is recommended to run the mkdocs server and view your pages in a browser.

## Prerequisites

- [Get Docker](https://docs.docker.com/get-docker/)
- [Get Task](https://taskfile.dev/installation/)

### NOTE to M1/M2 mac owners

If running on an arm64-based mac (M1 or M2, aka "Applie Silicon"), you may find this method quite slow. Until
multiarch docker images can be built and made available, you may wish to build your own via:

```bash
git clone git@github.com:squidfunk/mkdocs-material.git
docker build -t docker.io/squidfunk/mkdocs-material .
```
