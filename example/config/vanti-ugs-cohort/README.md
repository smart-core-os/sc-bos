## Vanti UGS Cohort

This setup is a cohort version of the Vanti UGS example. It contains an area controller, a hub and an edge dateway.
This is meant to be more realistic than the single edge gateway/area controller example in vanti-ugs and should 
help testing and pick up issues related to gateway/proxying.

## UI Config

The UI config in this has user accounts enabled unlike the vanti ugs example. 
The means that logged in users can have a role assigned, some features are hidden / disabled without a role.
The 'users.json' file has a simple account with username 'admin' and password 'admin' that has the 'admin' role assigned.
Adjust this role to test the UI with different permissions. The 'admin' role has access to all features, but you can create other roles with more limited access if needed.

## Running the example
To run the example, follow these steps:
`cd scripts`
`./run-vanti-ugs-cohort.sh`

This will start all 3 components (area controller, hub and edge gateway) in the correct order.
The script will also start a yarn dev server with the mode set to 'vanti-ugs-cohort' which will load the UI config from this example and connect to the running components.

### Fresh start
If you want to start fresh, you can run the script with `--clean` to remove the existing data before starting the components:
`./run-vanti-ugs-cohort.sh --clean`
This assumes you have the utility `psql` installed in the command line.