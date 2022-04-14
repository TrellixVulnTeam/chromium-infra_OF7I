FROM ubuntu:20.04

ENV DEBIAN_FRONTEND noninteractive

RUN apt-get update && \
    apt-get dist-upgrade -y && \
    apt-get install -y isc-dhcp-server rsync

ADD dhcpd.conf.keys /etc/dhcp/ddns-keys/dhcpd.conf.keys

RUN mkdir -p /tools/admin/etc
RUN chmod -R 777 /tools

RUN apt-get autoremove && \
    apt-get autoclean && \
    rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/* /var/cache/man \
      /usr/share/{doc,groff,info,lintian,linda,man} && \
    # Docker runs as 'chrome-bot' user on GCE bots, which is UID/GID 1000. We
    # need to write files to /etc/dhcp so move everything to GID 1000 writable.
    chown -R root:1000 /etc/dhcp && \
    chmod -R 775 /etc/dhcp

VOLUME /src

# Mounted from chrome-golo repo.
ENTRYPOINT ["/src/services/dhcpd/recipe_test.sh"]
