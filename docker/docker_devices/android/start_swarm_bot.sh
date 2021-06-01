#!/bin/bash -x

trap "exit 10" SIGUSR1

SWARM_DIR=/b/swarming
SWARM_ZIP=swarming_bot.zip

DEPOT_TOOLS_DIR=/b/depot_tools
DEPOT_TOOLS_URL="https://chromium.googlesource.com/chromium/tools/depot_tools.git"
DEPOT_TOOLS_REV="da3a29e13e816459234b0b08ed1059300bae46dd"

# Wait until this container has access to a device before starting the bot.
START=$(/bin/date +%s)
TIMEOUT=$((60*5))
while [ ! -d /dev/bus/usb ]
do
  now=$(/bin/date +%s)
  if [[ $((now-START)) -gt $TIMEOUT ]]; then
    echo "Timed out while waiting for an available device. Quitting early." 1>&2
    exit 1
  else
    echo "Waiting for an available usb device..."
    sleep 10
  fi
done

# Some chromium tests need depot tools.
mkdir -p $DEPOT_TOOLS_DIR
chown chrome-bot:chrome-bot $DEPOT_TOOLS_DIR
su -c "cd $DEPOT_TOOLS_DIR && \
       /usr/bin/git init && \
       /usr/bin/git remote add origin $DEPOT_TOOLS_URL ; \
       /usr/bin/git fetch origin $DEPOT_TOOLS_REV && \
       /usr/bin/git reset --hard FETCH_HEAD" chrome-bot

mkdir -p $SWARM_DIR
chown chrome-bot:chrome-bot $SWARM_DIR
cd $SWARM_DIR
rm -rf swarming_bot*.zip
su -c "/usr/bin/curl -sSLOJ $SWARM_URL" chrome-bot

echo "Starting $SWARM_ZIP"
# Run the swarming bot in the background, and immediately wait for it. This
# allows the signal trapping to actually work.
# Test out python3 on a single host on dev.
# TODO(crbug.com/1012230): Move all bots to python3.
# chromium-swarm-dev: build1-h9, build20-b4, build244-m4, build248-m4
# chromium-swarm: build2-h9
# chrome-swarming: build22-h7
if [[ "$(hostname -s)" =~ build("1-h9"|"20-b4"|"244-m4"|"248-m4"|"2-h9"|"22-h7")"--device"[1-4] ]]; then
  py_bin="/usr/bin/python3"
else
  py_bin="/usr/bin/python"
fi

su -c "${py_bin} $SWARM_ZIP start_bot" chrome-bot &
wait %1
exit $?
