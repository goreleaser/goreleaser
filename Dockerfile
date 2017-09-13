FROM scratch
COPY goreleaser /goreleaser
ENTRYPOINT ["/goreleaser"]

