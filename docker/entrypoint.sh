#!/bin/sh
set -e

# Fix ownership of mounted volumes if running as root
if [ "$(id -u)" = "0" ]; then
    chown -R vget:vget /home/vget/downloads /home/vget/.config/vget
    exec su-exec vget vget "$@"
else
    exec vget "$@"
fi
