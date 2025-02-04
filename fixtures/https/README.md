# Commands to generate certificates:

Commands are based on the following tutorial: https://medium.com/@harsha.senarath/how-to-implement-tls-ft-golang-40b380aae288

## Self-Signed CA:
    openssl req -new -newkey rsa:2048 -keyout ca.key -x509 -sha256 -days 999999 -out ca.crt
## Server Certificated based on self-Signed CA:
    openssl genrsa -out server.key 2048
    openssl req -new -key server.key -out server.csr -config server.cnf
    openssl req -noout -text -in server.csr
    openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out server.crt -days 999999 -sha256 -extfile server.cnf -extensions v3_ext
