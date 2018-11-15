[![pipeline status](https://git.f-i-ts.de/cloud-native/metal/metal-api/badges/master/pipeline.svg)](https://git.f-i-ts.de/cloud-native/metal/metal-api/commits/master)
[![coverage report](https://git.f-i-ts.de/cloud-native/metal/metal-api/badges/master/coverage.svg)](https://git.f-i-ts.de/cloud-native/metal/metal-api/commits/master)

# Metal API

Implementation of the *Metal API*

## Local development

If the netbox-API changes, you need an installation of `go-swagger` to create a client for accessing netbox.
Normally you do not need this, because the generated client is vendored and versioned.

To compile the service, simply call `make` if you want to force the compilation do a `make clean all`. In
most cases you also want a docker image so call `make clean all localbuild` which will create a new image
`registry.fi-ts.io/metal/metal-api` without a tag (aka `latest`).

To run the service, you need to have a clone of `metal-lab`. Enter the `provision/api` directory and do
a `docker-compose up -d && docker-compose logs -f`.

> Be sure to do a regularly `docker-compose pull` to have up-to-date images.

After building a new image for `metal-api` you simply do a `CTRL-C` and again `docker-compose up -d && docker-compose logs -f`.
This will take the new image for `metal-api`.
