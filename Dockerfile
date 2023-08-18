FROM alpine:3.18
RUN apk -U add ca-certificates
COPY bin/metal-api /metal-api
CMD ["/metal-api"]
