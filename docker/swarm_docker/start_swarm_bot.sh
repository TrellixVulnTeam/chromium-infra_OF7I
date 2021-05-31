#!/bin/bash -x

trap "exit 10" SIGUSR1

SWARM_DIR=/b/swarming
SWARM_ZIP=swarming_bot.zip

mkdir -p $SWARM_DIR
/bin/chown chrome-bot:chrome-bot $SWARM_DIR
cd $SWARM_DIR
rm -rf swarming_bot*.zip
/bin/su -c "/usr/bin/curl -sSLOJ $SWARM_URL" chrome-bot

echo "Starting $SWARM_ZIP"
# Run the swarming bot in the background, and immediately wait for it. This
# allows the signal trapping to actually work.
# TODO(crbug.com/1111688): Move all bots to python3.
if [[ "$(hostname -s)" = "build447-a9--000" ]]; then
  py_bin="/usr/bin/python3"
else
  py_bin="/usr/bin/python"
fi

/bin/su -c "${py_bin} $SWARM_ZIP start_bot" chrome-bot &
wait %1
exit $?
