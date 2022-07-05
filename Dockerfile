FROM metalstack/builder:latest as builder

FROM alpine:3.16
RUN apk -U add ca-certificates
COPY --from=builder /work/bin/metal-api /metal-api
CMD ["/metal-api --headscale-addr=host.docker.internal:50443 --headscale-api-key=b0AIgMcCMw.vydAHHSL8kV9ky25SAv-EQcaDN7GI18fwFMQka5qoys"]
