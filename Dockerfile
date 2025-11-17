FROM gcr.io/distroless/static-debian12:nonroot
COPY bin/metal-api /metal-api
CMD ["/metal-api"]
