#!/bin/bash

set -euo pipefail
IFS=$'\n\t'

echo "Starting"
read -p "Your name: " YOUR_NAME < /dev/tty;
echo "Hello, $YOUR_NAME"

sleep 0.5

read -p "Your age: " YOUR_AGE < /dev/tty;
echo "Your age is $YOUR_AGE"

sleep 0.3

echo "Let me list this directory before I go..."
ls
sleep 0.4

echo "Bye!"
