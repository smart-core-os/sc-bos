# Smart core API Docs
This folder holds the script for auto-generating the api documentation. This is done using GitHub actions, which also deploys the documentation to the docs repo for inclusion in the [docs site](https://smart-core-os.github.io).

## Updating

### Generating the MD
The docs are generated using the `protoc-gen-doc` protoc plugin, available here: 
https://github.com/pseudomuto/protoc-gen-doc  To install it locally, simply run:
```shell script
go get -u github.com/pseudomuto/protoc-gen-doc/cmd/protoc-gen-doc
```

If you need to individually manually generate the API docs for whatever reason, you'll need to run
the following from the root of this folder (i.e. `/docs`) for each file:
```shell script
protoc -I ../protobuf/ --doc_out ./api/traits --doc_opt=./markdown.tmpl,on_off.md ../protobuf/traits/on_off.proto
```

For convenience, a shell script is included to regenerate all packages (linux only):
```shell script
$ ./generate.sh
```

If you have added a new package, please add it to the generate script.
