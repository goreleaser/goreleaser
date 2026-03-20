FROM squidfunk/mkdocs-material:9@sha256:868ad4d39fb5865b72d00173ade00f4eae2b38dde7ff790a011cc44ce4a8ff8e
COPY requirements.txt .
RUN pip install -r requirements.txt
CMD ["serve", "--dev-addr=0.0.0.0:8000","--dirty"]
