In order for our builds to be hermetic (without network access) we configure rpm-prefetching.

If you want to prefetch more RPMs in order to install them during the build, you need to update the `rpms.in.yaml` file and follow [this doc](https://konflux.pages.redhat.com/docs/users/building/prefetching-dependencies.html#enabling-prefetch-builds-for-rpm) to generate the `rpms.lock.yaml` file.

If you want to better understand the `rpms.in.yaml` file you can look at the project's README [here](https://github.com/konflux-ci/rpm-lockfile-prototype/blob/main/README.md).
