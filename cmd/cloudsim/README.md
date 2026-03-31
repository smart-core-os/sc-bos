# cloudsim

`cloudsim` is a development and testing tool for an upcoming cloud-based config distribution system for 
Smart Core BOS. 

This is only intended for development and testing purposes. It is not production ready - it lacks authz/n and 
hardening for operating on the public internet.

It provides an HTTP JSON API. The API is implemented in `/internal/cloud/sim`; see the OpenAPI spec for an
endpoint list.

All data is stored in an SQLite database. See `/internal/cloud/sim/store` for the schema and access code.