# metal-api

![Build](https://github.com/metal-stack/metal-api/workflows/Build%20from%20master/badge.svg)
[![Slack](https://img.shields.io/badge/slack-metal--stack-brightgreen.svg?logo=slack)](https://metal-stack.slack.com/)
[![Go Report Card](https://goreportcard.com/badge/github.com/metal-stack/metal-api)](https://goreportcard.com/report/github.com/metal-stack/metal-api)
[![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/metal-stack/metal-api)
[![Docker Pulls](https://img.shields.io/docker/pulls/metalstack/metal-api.svg)](https://hub.docker.com/r/metalstack/metal-api/)

The metal-api is one of the major components of the metal-stack control plane. It is both the public interface for users to manage machines, networks, ips, and so forth and it is also the interface for metal-stack components running inside a partition.

The CLI tool for using the API is called `metalctl`. You can find this project [here](https://github.com/metal-stack/metalctl).
