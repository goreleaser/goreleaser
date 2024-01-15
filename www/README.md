# Documentation

Documentation is written in mkdocs and there are a few extensions that allow richer
authoring than markdown.

To iterate with documentation, therefore, it is recommended to run the mkdocs server and view your pages in a browser.

## Prerequisites

- [Get Docker](https://docs.docker.com/get-docker/)
- [Get Task](https://taskfile.dev/installation/)
- [Get MkDocs](https://www.mkdocs.org/user-guide/installation/)
  - [Get MkDocs Material](https://squidfunk.github.io/mkdocs-material/getting-started/#installation)
  - [Get MkDocs Redirect](https://github.com/mkdocs/mkdocs-redirects#installing)
  - [Get MkDocs Minify](https://github.com/byrnereese/mkdocs-minify-plugin#setup)
  - [Get MkDocs Include Markdown](https://github.com/mondeja/mkdocs-include-markdown-plugin#installation)
  - [Get MkDocs RSS](https://github.com/guts/mkdocs-rss-plugin#installation)

### NOTE to M1/M2 mac owners

If running on an arm64-based mac (M1 or M2, aka "Apple Silicon"), you may find this method quite slow. Until
multiarch docker images can be built and made available, you may wish to build your own via:

```bash
git clone git@github.com:squidfunk/mkdocs-material.git
docker build -t docker.io/squidfunk/mkdocs-material .
```

## Edit the docs

After installing mkdocs and extensions, build and run the documentation locally:

```sh
task docs:serve
```

The site will soon be available at http://0.0.0.0:8000 and update after changes.
