#!/bin/sh
echo -e "nameserver 127.0.0.11\nnameserver 8.8.8.8\nnameserver 223.5.5.5\n" > /etc/resolv.conf
/dashboard/app