FROM registry.fi-ts.io/cloud-native/go-builder:latest as builder
RUN make test-ci

FROM alpine:3.8
LABEL maintainer FI-TS Devops <devops@f-i-ts.de>
COPY --from=builder /work/bin/metal-api /metal-api
COPY --from=builder /work/generate/redoc.html /generate/redoc.html
CMD ["/metal-api"]
