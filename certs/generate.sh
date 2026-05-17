#!/usr/bin/env sh
# Regenerate all test certificates using cfssl.
# Prerequisites: cfssl and cfssljson must be on PATH.
#   go install github.com/cloudflare/cfssl/cmd/cfssl@latest
#   go install github.com/cloudflare/cfssl/cmd/cfssljson@latest
#
# The generated files are committed to the repository so that running the
# compose TLS tests does not require cfssl to be installed.

set -e
cd "$(dirname "$0")"

echo "==> CA"
cfssl gencert -initca ca-csr.json | cfssljson -bare ca

echo "==> Server (mqtt-tls broker)"
cfssl gencert -ca ca.pem -ca-key ca-key.pem \
  -config ca-config.json -profile server \
  server-csr.json | cfssljson -bare server

echo "==> Client: sensor-0"
cfssl gencert -ca ca.pem -ca-key ca-key.pem \
  -config ca-config.json -profile client \
  sensor-0-csr.json | cfssljson -bare sensor-0

echo "==> Client: sensor-1"
cfssl gencert -ca ca.pem -ca-key ca-key.pem \
  -config ca-config.json -profile client \
  sensor-1-csr.json | cfssljson -bare sensor-1

echo "==> Client: sensor-2"
cfssl gencert -ca ca.pem -ca-key ca-key.pem \
  -config ca-config.json -profile client \
  sensor-2-csr.json | cfssljson -bare sensor-2

# Remove the CSR files cfssl produces alongside the certs — not needed.
rm -f ca.csr server.csr sensor-0.csr sensor-1.csr sensor-2.csr

# Key files must be world-readable so the mosquitto container (which drops
# to the mosquitto user) can read them when mounted as a Docker volume.
chmod 644 ./*-key.pem

echo "Done. Generated files:"
ls -1 ./*.pem
