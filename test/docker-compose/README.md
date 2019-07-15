# docker-compose

This directory exists to run tests against the REST API of metal-api.

## Requirements

- The following projects are locally available:
  - `../../../metal-lab` [https://git.f-i-ts.de/cloud-native/metal/metal-lab](https://git.f-i-ts.de/cloud-native/metal/metal-lab)
  - `../../../metalctl` [https://git.f-i-ts.de/cloud-native/metal/metalctl](https://git.f-i-ts.de/cloud-native/metal/metalctl)

## Run Tests

1. Compile required binaries:
    - Metal API via `make` in `../../`
    - Metalctl via `make` in `../../../metalctl`
1. Start Metal Lab control plane via `make api` in `../../../metal-lab` as it holds the database and required K8s endpoints.
1. Run the compiled Metal API to test against it: `docker-compose up`.
1. Start Visual Studio Code to trigger REST calls:
    - Install Rest Client extension [https://marketplace.visualstudio.com/items?itemName=humao.rest-client](https://marketplace.visualstudio.com/items?itemName=humao.rest-client).
    - Provide Environment for the Rest Client in `vscode/userdata/User/settings.json`:

        ```json
            "rest-client.environmentVariables": {
                "$shared": {},
                "local-metal": {
                    "scheme": "http",
                    "host": "localhost:8080"
                },
            },
        ```

    - Activate the Rest Client environment "local-metal" by opening the Visual Studio Code "Command Palette" and     selecting "Rest Client: switch environment".
    - Trigger REST calls from interactive links in `../test/rest/*.rest` files.
