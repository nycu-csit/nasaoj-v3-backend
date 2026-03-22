#!/bin/sh
echo "Running inside container"
echo "System: $(grep -e '^NAME' /etc/os-release)"
echo "Hostname: $(hostname)"
echo "Working directory: $(pwd)"
echo "Success!"
