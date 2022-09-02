FROM metalstack/builder:latest as builder

FROM alpine:3.16
RUN apk -U add ca-certificates
COPY --from=builder /work/bin/metal-api /metal-api
CMD ["/metal-api", "--headscale-api-key=t-c59Xz83A.7gfBg50Cfl_YVkBlwHnduD-aJXBKKMaXRXtjvqO_aOo", "--headscale-addr=headscale.headscale:50443", "--headscale-cp-addr=http://headscale.172.17.0.1.nip.io:8080"]
