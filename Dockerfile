FROM golang:1.11

RUN apt-get update && \
    apt-get install -y --no-install-recommends apt-transport-https \
                                               curl \
                                               git \
                                               gnupg2 \
                                               rpm \
                                               software-properties-common && \
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg | apt-key add - && \
    apt-key fingerprint "9DC858229FC7DD38854AE2D88D81803C0EBFCD88" | grep "<docker@docker.com>" && \
    add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/debian $(lsb_release -cs) stable" && \
    apt-get update && \
    apt-get install -y --no-install-recommends docker-ce && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

ENTRYPOINT ["/entrypoint.sh"]
CMD [ "-h" ]

COPY entrypoint.sh /entrypoint.sh
COPY goreleaser /bin/goreleaser

RUN chmod +x /entrypoint.sh
