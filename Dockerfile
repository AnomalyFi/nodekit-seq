FROM ethereum/client-go:v1.13.0

RUN apk add --no-cache jq

COPY entrypoint-l1.sh /entrypoint.sh

VOLUME ["/db"]

ENTRYPOINT ["/bin/sh", "/entrypoint.sh"]
