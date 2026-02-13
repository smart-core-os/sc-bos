# cloudsim

`cloudsim` is a development and testing tool for an upcoming cloud-based config distribution system for 
Smart Core BOS. 

This is only intended for development and testing purposes. It is not production ready - it lacks authz/n and 
hardening for operating on the public internet.

It provides an HTTP JSON API. See the OpenAPI spec in `api/` for details.

All data is stored in an SQLite database. The schema is defined in migration files in `store/migrations/`.