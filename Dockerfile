FROM golang:1.11-stretch as builder
RUN apt update \
 && apt -y install make git
WORKDIR /app

# Install dependencies
COPY go.mod .
RUN go mod download

# Build
COPY .git ./.git
COPY cmd ./cmd
COPY pkg ./pkg
COPY Makefile ./Makefile
RUN make

FROM alpine:3.8
LABEL maintainer FI-TS Devops <devops@f-i-ts.de>
COPY --from=builder /app/cmd/maas-api/bin/maas-api /maas-api
CMD ["/maas-api"]
