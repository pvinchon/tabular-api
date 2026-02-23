#!/bin/bash

set -e

docker_compose_up() {
    echo "Create and start containers"

    docker compose up --build --detach --wait
}

docker_compose_down() {
    echo "Stop and remove containers"
    docker compose down
}

cd "$(dirname "$0")/.." || exit

docker_compose_up
trap docker_compose_down EXIT

output=$(curl -s http://localhost:8080)
if [[ "$output" == *"Hello, World!"* ]]; then
    echo "Test passed: Received expected output."
else
    echo "Test failed: Output did not match expected value."
    echo "Received output: $output"
    exit 1
fi