FROM alpine:3.18.5
COPY vault-plugin-manager /usr/local/bin/
USER 100:1000
ENTRYPOINT ["/usr/local/bin/vault-plugin-manager"]
