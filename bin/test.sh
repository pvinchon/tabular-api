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

test_playwright() {
    echo "Running Playwright tests..."

    cd tests/playwright
    npm install --silent
    npx playwright test
}

cd "$(dirname "$0")/.." || exit

docker_compose_up
trap docker_compose_down EXIT

test_playwright