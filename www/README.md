# Documentation

Documentation is written in mkdocs and there are a few extensions that allow richer
authoring than markdown.

To install them and iterate with documentation locally, [pipenv](https://pipenv.pypa.io/en/latest/) may be used to pull
down [mkdocs](https://www.mkdocs.org/) and its extensions

## Python and pip prerequisite

- [Install python3](https://www.python.org/downloads/) if not installed
- Install [pip](https://pip.pypa.io/en/stable/installation/)

## Installing pipenv

```bash
python3 -m pip install pipenv
```

## Installing mkdocs and its dependencies (extensions)

```bash
python3 -m pipenv sync
```

## Launching mkdocs to serve content locally for iteration

```bash
python3 -m pipenv run mkdocs serve
```

You should see something like

```
INFO     -  Documentation built in 4.63 seconds
INFO     -  [11:54:10] Watching paths for changes: 'docs', 'mkdocs.yml'
INFO     -  [11:54:10] Serving on http://127.0.0.1:8000/
```

Then, browse to the url listed <http://127.0.0.1:8000/> in this case to ensure your changes look good as you write them
(serve will force a refresh in the browser when it notices updates)
