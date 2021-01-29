#!/bin/sh
set -x

ME=`whoami`

ping example.com -c20 && \
    echo "==== $ME ====" && \
    ping example.net -c20 && \
    echo $1 && \
    echo "==== done! ===="