FROM metalstack/builder:latest as builder

FROM node:alpine as docbuilder
RUN npm install -g redoc-cli
COPY --from=builder /work/spec/metal-api.json /spec/metal-api.json
RUN redoc-cli bundle -o /generate/index.html /spec/metal-api.json

FROM alpine:3.12
RUN apk -U add ca-certificates
COPY --from=builder /work/bin/metal-api /metal-api
COPY --from=docbuilder /generate/index.html /generate/index.html
CMD ["/metal-api"]
