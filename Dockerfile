FROM golang:1.10
RUN apt-get update && \
	apt-get install -y rpm git && \
	rm -rf /var/lib/apt/lists/*
COPY goreleaser /goreleaser
ENTRYPOINT ["/goreleaser"]

