# Public TLS test fixtures

These certificates and the server private key are public, disposable test data.
The tests trust this CA only in an isolated client SSL context. Never install
this CA or use these credentials outside the loopback tests.

The RSA-2048 certificates use SHA-256 signatures and fixed validity dates from
2020-01-01 through 2120-01-01. The server certificate contains only the
`127.0.0.1` IP subject alternative name, so `localhost` exercises hostname
rejection. The CA private key is discarded after generation.

Generated with OpenSSL 4.0.3 using the commands below. OpenSSL is **not** needed
to run the tests; Python's standard-library `ssl` module reads the fixtures.
To regenerate, run these commands in this directory:

```sh
set -e
fixture_tmp=$(mktemp -d)
trap 'rm -rf "$fixture_tmp"' EXIT
openssl req -x509 -newkey rsa:2048 -noenc -sha256 \
  -subj '/CN=KFP readiness public test CA' \
  -not_before 20200101000000Z -not_after 21200101000000Z \
  -addext 'basicConstraints=critical,CA:TRUE' \
  -addext 'keyUsage=critical,keyCertSign,cRLSign' \
  -keyout "$fixture_tmp/ca.key" -out ca.pem
openssl req -x509 -newkey rsa:2048 -noenc -sha256 \
  -subj '/CN=KFP readiness public test server' \
  -CA ca.pem -CAkey "$fixture_tmp/ca.key" \
  -not_before 20200101000000Z -not_after 21200101000000Z \
  -addext 'basicConstraints=critical,CA:FALSE' \
  -addext 'keyUsage=critical,digitalSignature,keyEncipherment' \
  -addext 'extendedKeyUsage=serverAuth' \
  -addext 'subjectAltName=IP:127.0.0.1' \
  -keyout server.key -out server.pem
openssl verify -CAfile ca.pem -purpose sslserver \
  -verify_ip 127.0.0.1 server.pem
```
