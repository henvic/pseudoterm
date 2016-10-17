#!/bin/bash

# this mock is used on TestTerminalWithStoryShouldNotBlock
# which mixes both Story and writing to it directly

set -euo pipefail
IFS=$'\n\t'

function checkCONT() {
  if [[ $CONT != "y" && $CONT != "yes" ]]; then
    exit 1
  fi
}

read -p "Continue? [no]: " CONT < /dev/tty;
checkCONT

echo "Starting"
read -p "Your name: " YOUR_NAME < /dev/tty;
echo "Your name is $YOUR_NAME"

sleep 0.05

read -p "Your age: " YOUR_AGE < /dev/tty;
echo "Your age is $YOUR_AGE"

sleep 0.1

read -p "Avoid killing itself? [no]: " CONT < /dev/tty;
checkCONT

echo "Bye!"
