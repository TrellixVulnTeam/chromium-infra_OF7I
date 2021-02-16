FROM ubuntu:bionic

ENV DEBIAN_FRONTEND noninteractive
ENV CHROMIUM_REV 4f36f9c9d232eeb83825f6f5597ae17dd6786445

RUN /usr/bin/apt-get update
RUN /usr/bin/apt-get -y dist-upgrade
RUN /usr/bin/apt-get -y install curl iputils-ping locales lsb-release lsof \
      net-tools netcat-openbsd parted python rsync sshpass sudo udev usbutils \
      wget

# Install tzdata here rather than as part of install-build-deps.sh since the
# DEBIAN_FRONTEND=noninteractive env var doesn't appear to apply to
# install-build-deps.sh and the package ends up waiting on user input.
RUN /usr/bin/apt-get -y install tzdata

# Every Chrome test today should be using the hermetic JDK fetched as part of
# a Chrome checkout. However, prior to crrev.com/c/2660219, some tests were
# depending on a JDK insalled in the system. And since pinpoint is still testing
# revisions prior to that change, we have to oblige it and continue installing
# JDKs in the system.
# FIXME: Remove this once pinpoint no longer tests revisions prior to
# crrev.com/c/2660219. Likely sometime in 2022.
RUN /usr/bin/apt-get -y install default-jdk

RUN echo 'ttf-mscorefonts-installer msttcorefonts/accepted-mscorefonts-eula select true' \
      | /usr/bin/debconf-set-selections
RUN echo 'msttcorefonts msttcorefonts/defoma select note' \
      | /usr/bin/debconf-set-selections

RUN /bin/mkdir /tmp/setup-deps
RUN /usr/bin/wget -O- \
      "https://chromium.googlesource.com/chromium/src/+/${CHROMIUM_REV}/build/install-build-deps.sh?format=TEXT" \
        | /usr/bin/base64 -d \
        > /tmp/setup-deps/install-build-deps.sh
RUN /usr/bin/wget -O- \
      "https://chromium.googlesource.com/chromium/src/+/${CHROMIUM_REV}/build/install-build-deps-android.sh?format=TEXT" \
        | /usr/bin/base64 -d \
        > /tmp/setup-deps/install-build-deps-android.sh
RUN /bin/chmod a+x /tmp/setup-deps/install-build-deps*.sh
# This runs install-build-deps.sh as well.
RUN /bin/bash -x /tmp/setup-deps/install-build-deps-android.sh

# Cleanup to shrink image sizes.
RUN /usr/bin/find /var/lib/apt/lists /tmp /var/tmp /usr/share/info \
      /usr/share/lintian /usr/share/man -maxdepth 1 -mindepth 1 -exec rm -rf {} \;
RUN /usr/bin/find /usr/share/doc -depth -type f ! -name copyright -exec rm -rf {} \;
RUN /usr/bin/find /usr/share/doc -depth -empty -exec rm -rf {} \;

RUN /bin/mkdir /opt/swarming_bootstrap
ADD start_swarm_bot.sh /opt/swarming_bootstrap/start_swarm_bot.sh
# /sbin/shutdown is a symlink to /bin/systemctl prior to bionic, and ADDing a
# file at a symlink replaces the target of the symlink. So remove the symlink
# first before adding shutdown.sh so we don't end up replacing /bin/systemctl.
RUN /bin/rm -f /sbin/shutdown
ADD shutdown.sh /sbin/shutdown
RUN /bin/chmod 0755 /sbin/shutdown
RUN echo "%chrome-bot ALL = NOPASSWD: /sbin/shutdown" > /etc/sudoers.d/chrome-bot

CMD ["/bin/bash", "/opt/swarming_bootstrap/start_swarm_bot.sh"]
