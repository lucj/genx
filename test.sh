#!/usr/bin/env bash
set -euo pipefail

PROFILES=(
  test-webhook
  test-cloudevent
  test-mqtt
  test-mqtt-auth
  test-mqtt-tls
  test-mqtt-mtls
  test-nats
  test-nats-userpass
  test-nats-token
  test-kafka
)

passed=()
failed=()

for profile in "${PROFILES[@]}"; do
  echo "=== Running $profile ==="
  if docker compose --profile "$profile" up \
      --build \
      --abort-on-container-exit \
      --exit-code-from "$profile" 2>&1; then
    passed+=("$profile")
  else
    failed+=("$profile")
  fi
  docker compose --profile "$profile" down -v 2>/dev/null || true
  sleep 2
done

echo ""
echo "=== Results ==="
for p in "${passed[@]}"; do echo "  PASS  $p"; done
for f in "${failed[@]}"; do echo "  FAIL  $f"; done

[[ ${#failed[@]} -eq 0 ]]
