FROM golang:1.12-alpine as builder

LABEL maintainer="Olaoluwa Osuntokun <laolu@lightning.engineering>"

# Grab and install the latest version of roasbeef's fork of acmd and all
# related dependencies.
WORKDIR $GOPATH/src/github.com/Actinium-project/acmd
RUN apk add --no-cache git \
&&  git clone https://github.com/Actinium-project/acmd ./ \
&&  go get -u github.com/Masterminds/glide \
&&  glide install \
&&  go install . ./cmd/ltcctl ./cmd/gencerts

# Start a new image
FROM alpine as final

# Expose mainnet ports (server, rpc)
EXPOSE 9333 9334

# Expose testnet ports (server, rpc)
EXPOSE 19334 19335

# Expose simnet ports (server, rpc)
EXPOSE 18555 18556

# Copy the compiled binaries from the builder image.
COPY --from=builder /go/bin/ltcctl /bin/
COPY --from=builder /go/bin/acmd /bin/
COPY --from=builder /go/bin/gencerts /bin/

COPY "start-ltcctl.sh" .
COPY "start-acmd.sh" .

RUN apk add --no-cache \
    bash \
    ca-certificates \
&&  chmod +x start-ltcctl.sh \
&&  chmod +x start-acmd.sh \
&&  mkdir "/rpc" "/root/.acmd" "/root/.ltcctl" \
&&  touch "/root/.acmd/acmd.conf" \
# "ltcctl" and "lnd" to "acmd" over docker links.
&&  "/bin/gencerts" --host="*" --directory="/rpc" --force

# Create a volume to house pregenerated RPC credentials. This will be
# shared with any lnd, btcctl containers so they can securely query acmd's RPC
# server.
# You should NOT do this before certificate generation!
# Otherwise manually generated certificate will be overridden with shared
# mounted volume! For more info read dockerfile "VOLUME" documentation.
VOLUME ["/rpc"]
