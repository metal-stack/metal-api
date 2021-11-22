FROM metalstack/builder:latest as builder

FROM alpine:3.14
RUN apk -U add ca-certificates
COPY --from=builder /work/bin/metal-api /metal-api
CMD ["/metal-api"]
