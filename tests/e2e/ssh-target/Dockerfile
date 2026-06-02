FROM debian:bookworm-slim

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        bash \
        coreutils \
        gawk \
        iproute2 \
        openssh-server \
        procps \
        util-linux \
    && rm -rf /var/lib/apt/lists/*

RUN mkdir -p /run/sshd \
    && useradd -m -s /bin/bash monitor \
    && printf 'monitor:monitor\n' | chpasswd

COPY entrypoint.sh /usr/local/bin/remote-monitor-e2e-entrypoint
RUN chmod +x /usr/local/bin/remote-monitor-e2e-entrypoint

EXPOSE 22

ENTRYPOINT ["/usr/local/bin/remote-monitor-e2e-entrypoint"]
