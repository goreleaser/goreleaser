FROM squidfunk/mkdocs-material:9@sha256:1c30983ef0681684d07a90f4f34a2ff8adf7aaa0bfc1c0d518873bb5e7912f37
COPY requirements.txt .
RUN pip install -r requirements.txt
CMD ["serve", "--dev-addr=0.0.0.0:8000","--dirty"]
