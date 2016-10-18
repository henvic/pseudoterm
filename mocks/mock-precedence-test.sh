#!/bin/bash

set -euo pipefail
IFS=$'\n\t'

echo "Starting"
read -p "Your name: " YOUR_NAME < /dev/tty;
echo "Your name is $YOUR_NAME"

sleep 0.05

read -p "Your age: " YOUR_AGE < /dev/tty;
echo "Your age is $YOUR_AGE"

sleep 0.1

read -p "Do you want a peach? " YOUR_OK < /dev/tty;
echo "peach: $YOUR_OK"

read -p "Random: $RANDOM: " YOUR_NUM < /dev/tty;
echo "num: $YOUR_NUM"
echo "Bye!"
