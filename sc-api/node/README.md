# Smart Core API packaged for npm
This is the npm packaged version of the Smart Core API definitions (https://github.com/smart-core-os/sc-api). If you're
working with Smart Core using node, you probably want the `@smart-core-os/server` or `@smart-core-os/client` packages.

This package is just a copy of the core `.proto` files, with a `package.json` attached for publishing.

Note: Currently, the local `.proto` files aren't committed back to the git repo to avoid duplication, however this may 
change if versioning becomes an issue.

## Using
These `.proto` files are intended for use with the `@grpc/proto-loader` package that will dynamically generate the JS
code at runtime.

## Updating
If you have made changes to the API definition files and need to re-publish the package, you'll need to do the following:
 
 1. Update the version number in `package.json`
 2. Run `npm run gen` - this will copy all of the proto files from the protobuf folder into the local `./proto` folder. 
    At this point you can prune-out any files that aren't ready for publishing yet. 
 3. Then run `npm publish` to publish everything to npm
