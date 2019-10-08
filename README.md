[![pipeline status](https://git.f-i-ts.de/cloud-native/metal/metal-api/badges/master/pipeline.svg)](https://git.f-i-ts.de/cloud-native/metal/metal-api/commits/master)
[![coverage report](https://git.f-i-ts.de/cloud-native/metal/metal-api/badges/master/coverage.svg)](https://git.f-i-ts.de/cloud-native/metal/metal-api/commits/master)

# Metal API

Implementation of the *Metal API*

## Local development

Local Development is supported by running the environment in a local minikube.

### Preparation

* [minikube](https://github.com/kubernetes/minikube/releases)
* [helm](https://github.com/helm/helm/releases/) - helm 3 beta 3 works like a charm
* [kubefwd](https://github.com/txn2/kubefwd/releases)

Hint: kubefwd must be executed with root privileges, so move kubefwd to `/usr/local/bin`, `chown root:root`, and set SUID with `chmod +s`


### Deploy metal-api environment

Deploy 

```
make localkube-install
```

Start kubefwd

```
make local-forward
```

Test with HMAC

```
METALCTL_URL=http://metal-api:8080 METALCTL_HMAC=must-be-changed metalctl machine ls
```

Test with Token

```
METALCTL_URL=http://metal-api:8080 metalctl login
METALCTL_URL=http://metal-api:8080 metalctl machine ls
```

### Build and push metal-api

Build the metal-api docker-container and restarts the metal-api pod. **You have to do this once to see your local version of metal-api!**

```
make localbuild-push
```

You have to restart kubefwd. 

### Remove metal-api and environment deployment 

```
helm uninstall rethink metal
```

Please wait some time before you retry installation again, because the PVCs need some time to vanish.