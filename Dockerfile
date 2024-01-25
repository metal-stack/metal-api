FROM alpine:3.19
RUN apk -U add ca-certificates
COPY bin/metal-api /metal-api
CMD ["/metal-api"]
