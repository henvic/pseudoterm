#!/bin/bash

set -euo pipefail
IFS=$'\n\t'

echo "This is a slow-running version of complex.sh"
sleep 0.4
echo "It makes complex.go time out and never ends execution."
sleep 0.5
echo "This can be auto-executed by running \"go run slow-complex.go\""
sleep 0.6
echo "Or executed manually with ./slow-complex.sh"
sleep 0.7
echo "Starting"
sleep 0.8
read -p "Your name: " YOUR_NAME < /dev/tty;
echo "Your name is $YOUR_NAME"

sleep 0.9

read -p "Your age: " YOUR_AGE < /dev/tty;
echo "Your age is $YOUR_AGE"

sleep 1

read -p "Do you want a peach? " YOUR_OK < /dev/tty;
echo "peach: $YOUR_OK"

sleep 1.1

read -p "Random: $RANDOM: " YOUR_NUM < /dev/tty;
sleep 1.2
echo "num: $YOUR_NUM"
echo "Bye!"
