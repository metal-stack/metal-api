FROM gcr.io/distroless/static-debian13:nonroot
COPY bin/metal-api /metal-api
CMD ["/metal-api"]
