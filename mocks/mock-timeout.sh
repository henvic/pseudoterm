#!/bin/bash

set -euo pipefail
IFS=$'\n\t'

echo "Starting"
read -p "Your name: " YOUR_NAME < /dev/tty;
echo "Your name is $YOUR_NAME"

# timeout before showing "Your age: "
sleep 10

read -p "Your age: " YOUR_AGE < /dev/tty;
echo "Your age is $YOUR_AGE"

echo "Bye!"
