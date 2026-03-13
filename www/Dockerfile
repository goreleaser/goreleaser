FROM squidfunk/mkdocs-material:9@sha256:c3739993c3e7c92cfe649ba03cbc04f4fa9b1497886abd16267eb413cda8aecf
COPY requirements.txt .
RUN pip install -r requirements.txt
CMD ["serve", "--dev-addr=0.0.0.0:8000","--dirty"]
