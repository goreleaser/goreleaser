FROM squidfunk/mkdocs-material:9@sha256:8f41b6089700e1c32212c3857936f14e88a3306a35be4ffd1826420e2f3e4197
COPY requirements.txt .
RUN pip install -r requirements.txt
CMD ["serve", "--dev-addr=0.0.0.0:8000","--dirty"]
