FROM ubuntu:16.04

ENV DEBIAN_FRONTEND noninteractive

RUN apt-get update && \
    apt-get dist-upgrade -y && \
    apt-get install -y isc-dhcp-server rsync

ADD dhcpd.conf.keys /etc/dhcp/dhcpd.conf.keys
RUN mkdir -p /tools/admin/etc
RUN chmod -R 777 /tools

RUN apt-get autoremove && \
    apt-get autoclean && \
    rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/* /var/cache/man \
      /usr/share/{doc,groff,info,lintian,linda,man}

VOLUME /src

# Mounted from chrome-golo repo.
ENTRYPOINT ["/src/services/dhcpd/recipe_test.sh"]
