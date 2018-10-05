FROM golang:1.11
RUN apt-get update && \
	apt-get install -y --no-install-recommends rpm git apt-transport-https curl gnupg2 software-properties-common && \
	curl -fsSL https://download.docker.com/linux/debian/gpg | apt-key add - && \
	add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/debian $(lsb_release -cs) stable" && \
	apt-get update && \
	apt-get install -y --no-install-recommends docker-ce &&\
	rm -rf /var/lib/apt/lists/*
COPY goreleaser /bin/goreleaser
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh
ENTRYPOINT ["/entrypoint.sh"]
CMD [ "-h" ]
