# Ubuntu + Docker CE + Go for local e2e via Docker-in-Docker (see run.sh).
FROM ubuntu:24.04

ARG GO_VERSION=1.27.2
ARG TARGETARCH=amd64

ENV DEBIAN_FRONTEND=noninteractive \
	PATH=/usr/local/go/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin \
	GOTOOLCHAIN=local

RUN apt-get update \
	&& apt-get install -y --no-install-recommends \
		ca-certificates \
		curl \
		git \
		gnupg \
		iptables \
		iproute2 \
		kmod \
		procps \
		uidmap \
	&& install -m 0755 -d /etc/apt/keyrings \
	&& curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc \
	&& chmod a+r /etc/apt/keyrings/docker.asc \
	&& echo \
		"deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu \
		$(. /etc/os-release && echo "${VERSION_CODENAME}") stable" \
		> /etc/apt/sources.list.d/docker.list \
	&& apt-get update \
	&& apt-get install -y --no-install-recommends \
		docker-ce \
		docker-ce-cli \
		containerd.io \
		docker-buildx-plugin \
		docker-compose-plugin \
	&& rm -rf /var/lib/apt/lists/*

RUN curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-${TARGETARCH}.tar.gz" \
	| tar -C /usr/local -xz \
	&& go version

WORKDIR /src

COPY entrypoint.sh /usr/local/bin/apinfra-e2e-entrypoint
RUN chmod 0755 /usr/local/bin/apinfra-e2e-entrypoint

ENTRYPOINT ["/usr/local/bin/apinfra-e2e-entrypoint"]
