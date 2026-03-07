FROM squidfunk/mkdocs-material:9@sha256:f4332a8e8895aa303bf7f3ceede749980a98cf79f5d87ad567b4d41f037b0101
COPY requirements.txt .
RUN pip install -r requirements.txt
CMD ["serve", "--dev-addr=0.0.0.0:8000","--dirty"]
