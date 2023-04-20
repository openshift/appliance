# Run local registry image
mkdir -p {{.RegistryDataPath}}
podman rm registry --force
podman run --privileged -d --name registry \
    -p 5000:5000 -p 443:5000 \
    -v {{.RegistryDataPath}}:/var/lib/registry --restart=always \
    -e REGISTRY_HTTP_ADDR=0.0.0.0:5000 \
    docker.io/library/registry:2 