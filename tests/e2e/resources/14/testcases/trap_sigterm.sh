#!/usr/bin/env bash

trap 'echo "SIGTERM_RECEIVED"; exit 0' SIGTERM

echo "sleeping"
sleep 60
