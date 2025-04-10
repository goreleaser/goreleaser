# Documentation

Documentation is written in mkdocs and there are a few extensions that allow
richer authoring than markdown.

To iterate with documentation, therefore, it is recommended to run the mkdocs
server and view your pages in a browser.

To run the docs locally, do:

```bash
docker build -t material-mkdocs ./www
docker run --rm -p 8000:8000 -v ./www:/docs material-mkdocs
```

The site should soon be available at [http://localhost:8000] and
auto-update after most changes.

> [!TIP]
> If you use `task`, you can also run `task docs:serve` instead.
