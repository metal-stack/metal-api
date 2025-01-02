FROM gcr.io/distroless/static-debian12
COPY bin/metal-api /metal-api
CMD ["/metal-api"]
