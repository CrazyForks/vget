## Build Docker Image

### Build vget base image

```shell
ocker build --no-cache -f docker/vget-base/Dockerfile -t ghcr.io/guiyumin/vget-base:latest .
```

### CPU version (no models bundled - downloads on first use)

```shell
docker build -f docker/vget/Dockerfile -t vget:latest .
```

### CPU version with models pre-bundled

```shell
docker build -f docker/vget/Dockerfile -t vget:small --build-arg MODEL_VARIANT=small .
```

```shell
docker build -f docker/vget/Dockerfile -t vget:medium --build-arg MODEL_VARIANT=medium .
```

### CUDA/GPU version

```shell
docker build -f docker/vget/Dockerfile -t vget:cuda --build-arg ENABLE_CUDA=true .
```
