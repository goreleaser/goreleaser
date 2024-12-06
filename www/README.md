# Documentation

Documentation is written in mkdocs and there are a few extensions that allow richer
authoring than markdown.

To iterate with documentation, therefore, it is recommended to run the mkdocs server and view your pages in a browser.

## Setup dev environment

The basic pre-requisite is [Task](https://taskfile.dev/installation/).
For material-mkdocs itself, you can either do nix or Docker, as below.

### With nix

If you have nix, you can run:

```bash
nix develop .#docs -c task docs:serve
```

To drop into a shell with all the needed dependencies.

### With Docker

```bash
cd ./www
docker build -t material-mkdocs
docker run --rm -it -p 8000:8000 -v .:/docs material
```

---

In both cases, the site should soon be available at http://localhost:8000 and
auto-update after most changes.
