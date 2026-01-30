# Smart Core Demos

This demo directory contains a collection of demos that showcase the capabilities of Smart Core.

In each folder, there is a demo for a specific site, each with its own config and ui assets. This readme will use the
`vanti-ugs` demo as an example to explain how to run the demos. `vanti-ugs` is the Vanti Birmingham office instance of
Smart Core.

## Building and deploying the demos

Each folder has a `build.sh` script that builds the demo in that folder, the output of which will be one or more
container images.
The demos do not need to be built all the time, only when a new release of the demo is required.
The demos also contain a compose file that pulls all the infrastructure needed to run the demo into one place.
The images are then pushed to a container registry, so they can be accessed from anywhere the demo needs to be set up.

The demo images are built for multiple platforms, so they can run on both x86 and ARM architectures.
To enable this in Docker Desktop,
*Settings -> General -> Use containerd for pulling and storing images -> Enable*.

The images are published
to [GitHub Container Registry](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry),
which requires a Personal Access Token (PAT) with the `write:packages` scopes.
Use [this link](https://github.com/settings/tokens/new?scopes=write:packages) to set on up with the correct permissions.
Don't forget to **authorise the PAT with our organisations**.
Once you have the PAT, you can log in to the GitHub Container Registry using the following command:

```
$ podman login ghcr.io --username <your-github-username>
Password: <your-github-pat>
Login Succeeded!
```

Deploying the demo involves publishing the multi-platform images (aka manifests) to the GitHub Container Registry using
commands like:

```shell
GIT_VERSION=$(git describe --tags --always)
podman manifest push --all demo-ugs-sc-bos:$GIT_VERSION docker://ghcr.io/smart-core-os/sc-bos/demo-ugs-sc-bos:$GIT_VERSION
podman manifest push --all demo-ugs-seed-db:$GIT_VERSION docker://ghcr.io/smart-core-os/sc-bos/demo-ugs-seed-db:$GIT_VERSION
```

The build script should output the correct commands to run, so you can just copy and paste them.

Finally, the compose file is using tagged images in order to control when the demo app is updated.
You will need to update the tags mentioned in that file to the same as the version you just pushed to the registry.

## Running the demo

The demo runs a local instance of Smart Core and related infrastructure, including a db.
We've packaged these up into a Compose File, which is a convenient way to run multiple containers together.
Download 
the [docker-compose-ugs.yml](https://raw.githubusercontent.com/smart-core-os/sc-bos/main/demo/vanti-ugs/docker-compose-ugs.yml)
file to your machine.

To run the demo, you need to have Docker/Podman installed on your machine, for non-devs we recommend the Desktop
versions of each.
The desktop apps will have a way to add a container file to the application and run it.
For podman you do this by clicking on the Containers tab, then clicking on the `+ Create` button and selecting the
compose file `docker-compose-ugs.yml` in the file picker dialog.
Also, the Docker/Podman daemon must be running, on Windows you can just press the Windows key and type docker and start 
Docker/Podman desktop.

To start the demo using the command line, you will need to have Docker or Podman installed and running and can then run
the following command in the location where you downloaded the `docker-compose-ugs.yml` file:

```shell
# Or docker...
podman compose -f docker-compose-ugs.yml up -d
```

Then you can access the site at `https://localhost:8443/` in your browser. (for the Vanti example, for any other
examples refer to the `listenHTTPS` setting in the relevant system.conf.json)

You can also use this demo to test against the API, for example, to get the list of children of the root node,
run the `client-parent` tool in `cmd/tools` and it will print the devices in `vanti-ugs`
set the insecure-skip-verify flag to true as the demo uses a self-signed certificate.

### Connecting UIs to the sc-bos Backend

You may need to connect a UI running in the browser to the sc-bos instance(s) running inside docker.
For this, I have created a script in the vanti-ugs directory (which should really only be executed on MacOS laptops).
This script will copy the self-signed certificate file from the docker container running the vanti-ugs sc-bos instance.
It will then add the self-signed certificate file to the MacOS System Keychain (root) and trust it.

#### Trust the self-signed certificates

Execute with `sh demo/vanti-ugs/scripts/trust-ugs.sh`.
Restart your browser, and you should find you can connect to the sc-bos instance via grpc-web.