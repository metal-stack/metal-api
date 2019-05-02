FROM registry.fi-ts.io/cloud-native/go-builder:latest as builder
RUN make test-ci

FROM letsdeal/redoc-cli:latest as docbuilder
COPY --from=builder /work/spec/metal-api.json /spec/metal-api.json
RUN redoc-cli bundle -o /generate/redoc.html /spec/metal-api.json

FROM alpine:3.8
LABEL maintainer FI-TS Devops <devops@f-i-ts.de>
COPY --from=builder /work/bin/metal-api /metal-api
COPY --from=docbuilder /generate/redoc.html /generate/redoc.html
CMD ["/metal-api"]
