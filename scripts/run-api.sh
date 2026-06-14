#!/usr/bin/env bash
set -a
source "$(dirname "$0")/../.env"
set +a
cd "$(dirname "$0")/../backend"
exec ./server
