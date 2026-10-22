# ac-artifact-organization acceptance criteria

# scenario 1: Default artifact locations
Given the user runs `iso-builder build` without `--working-dir`
And `$XDG_CACHE_HOME` is not set
When the build completes successfully
Then the final ISO is written to the current directory as `agent.<arch>.iso`
And transient build artifacts are stored under `./.iso-builder`
And cached release artifacts are stored under `$HOME/.cache/iso-builder/release-<version>`

# scenario 2: Custom working directory
Given the user runs `iso-builder build --working-dir /my/builds/sno`
When the build completes successfully
Then the final ISO is written to `/my/builds/sno/agent.<arch>.iso`
And transient build artifacts are stored under `/my/builds/sno/.iso-builder`
And cached release artifacts remain under `$XDG_CACHE_HOME/iso-builder/release-<version>`

# scenario 3: Custom XDG_CACHE_HOME
Given the user has set `$XDG_CACHE_HOME` to an absolute path (e.g. /custom/cache)
When the user runs `iso-builder build`
Then cached release artifacts are stored under `/custom/cache/iso-builder/release-<version>`

# scenario 4: Relative XDG_CACHE_HOME is ignored
Given the user has set `$XDG_CACHE_HOME` to a relative path (e.g. relative/cache)
When the user runs `iso-builder build`
Then the relative path is ignored and the default `$HOME/.cache/iso-builder` is used

# scenario 5: Per-release cache structure
Given the user runs a build for OCP version `x.y.z`
When the cache directory for that release is created
Then it is located at `<cache-base>/release-x.y.z`
And it contains subdirectories `tools/, images/, registry/, iso/`

# scenario 6: Cache reuse across builds
Given the user has already built an ISO for OCP version `x.y.z`
When the user runs another build for the same version with a different `--working-dir`
Then the cached release artifacts under `<cache-base>/release-x.y.z` are reused
And only the working directory contains new per-build artifacts
