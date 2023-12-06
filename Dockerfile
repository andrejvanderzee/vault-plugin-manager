FROM golang:alpine as builder
RUN mkdir /build
ADD . /build/
WORKDIR /build
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o vault-plugin-manager .

FROM alpine:3.18.5
COPY --from=builder /build/vault-plugin-manager /usr/local/bin/vault-plugin-manager
RUN addgroup --gid 1000 vault && \
    adduser --uid 100 --system -g vault vault
USER 100
ENTRYPOINT ["/usr/local/bin/vault-plugin-manager"]
