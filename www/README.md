# Documentation

Documentation is written in mkdocs and there are a few extensions that allow richer
authoring than markdown.

To iterate with documentation, therefore, it is recommended to run the mkdocs server and view your pages in a browser.

## Prerequisites

- [Get Task](https://taskfile.dev/installation/)
- [Get MkDocs](https://www.mkdocs.org/user-guide/installation/)
  - [Get MkDocs Material](https://squidfunk.github.io/mkdocs-material/getting-started/#installation)
  - [Get MkDocs Redirect](https://github.com/mkdocs/mkdocs-redirects#installing)
  - [Get MkDocs Minify](https://github.com/byrnereese/mkdocs-minify-plugin#setup)
  - [Get MkDocs Include Markdown](https://github.com/mondeja/mkdocs-include-markdown-plugin#installation)
  - [Get MkDocs RSS](https://github.com/guts/mkdocs-rss-plugin#installation)

### With nix

If you have nix installed, you can:

```bash
nix develop .#docs
```

To drop into a shell with all the needed dependencies.

### With Docker

```bash
cd ./www
docker build -t material-mkdocs
docker run --rm -it -p 8000:8000 -v .:/docs material
```

## Edit the docs

After installing mkdocs and extensions, build and run the documentation locally:

```sh
task docs:serve
```

The site will soon be available at http://localhost:8000 and
auto-update after changes.
