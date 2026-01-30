# Creating Releases

Releases are created automatically by GitHub Actions when a tag is pushed to the repository. The tag should be of the
form `v0.YYYY.R` (e.g. `v0.2024.1`). The 0 prefix is used to indicate the project is still under development and 
not bound by any compatibility guarantees. The `YYYY` is the year, and `R` is the release number for that year.

## Testing releases

You can test the release process locally using [act](https://github.com/nektos/act). This will run mostly the same steps 
as the GitHub Actions workflow, but locally without actually releasing/publishing to GitHub. You will need to have 
Docker installed, the `act` binary, and a file containing the secrets required for the build (the example below uses 
`.secret/act.secrets`):

```
GITHUB_TOKEN=<github personal access token, should start `ghp_`>
GO_MOD_TOKEN=<copy of GITHUB_TOKEN>
```

You can then run the following commands to perform the build of the binaries, UIs, and Docker images:
```shell
act --secret-file .secret/act.secrets --artifact-server-path /tmp/artifacts --pull false -j build-sc-bos
act --secret-file .secret/act.secrets --artifact-server-path /tmp/artifacts --pull false -j build-ops-ui
act --secret-file .secret/act.secrets --artifact-server-path /tmp/artifacts --pull false -j build-docker
``` 

_// todo: fix this issue!_\
Note: there is some permissions issue with the `build-docker` job that means it doesn't work with `act` - it's not clear 
why, but it's not a big deal as the Docker image is built by GitHub Actions anyway. However, it's useful to be able to 
test the build locally, so the manual steps to build the Docker image are as follows:

```shell
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o .build/sc-bos github.com/smart-core-os/sc-bos/cmd/bos
cd ui/ops
yarn install && yarn run build
cd ../..
mv ui/ops/dist .build/ops-ui
docker build -t ghcr.io/smart-core-os/sc-bos:vTest .
```

#### Gotchas:
- Using the manual compile method, if you have an `env.local` file for the ops-ui you'll need to comment out any overrides you have set
