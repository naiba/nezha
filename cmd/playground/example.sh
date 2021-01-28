#!/bin/sh
set -x

ME=`whoami`

ping example.com -c3 && \
    echo "==== $ME ====" && \
    ping example.net -c3 && \
    echo $1 && \
    echo "==== done! ===="