## Vanti UGS Demo

This is a demo of the Smart Core system running the Vanti Birmingham office instance. It contains the floorplans for the
UGS office and also the sensors that are in the office. They are using the mock driver in the demo so we can run this
without any hardware.

There are 2 Dockerfiles, one that starts Smart Core and one that seeds the database with the data for the UGS office.
The data is seeded for the past 31 days, in order for the OPS UI to show some data.

NOTE: the database seeder takes a few minutes to run and populate all the data. 
After first starting the demo, wait 10 minutes before showing dashboards.

Refer the top README in the `demo` directory for instructions on how to build the containers and push to a registry.
This step only needs to be done once, or if the demo is updated.

Then it is just a case of running the docker-compose file.

Each time the demo images are built they copy the current version of the UGS example config and build the UI apps.
To have the demo pick up the latest config or UI changes, re-run the build and update the version tags in the compose
file.

## Weather widget (OpenWeatherMap API key)

The overview dashboard includes a live weather widget that requires an OpenWeatherMap API key.
The key is never baked into the image — it is injected at container startup via an environment variable.

### Setup

1. Get a free API key at https://openweathermap.org/api (the "Current Weather Data" free tier is sufficient).
2. Copy `.env.example` to `.env` in this directory:
   ```
   cp demo/vanti-ugs/.env.example demo/vanti-ugs/.env
   ```
3. Edit `.env` and set your key:
   ```
   OPENWEATHERMAP_API_KEY=your_key_here
   ```
4. Docker Compose picks up `.env` automatically — no other changes needed.

The `.env` file is gitignored and will never be committed. If the variable is not set the weather widget will
simply fail to load data; the rest of the demo is unaffected.
