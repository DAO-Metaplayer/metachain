# Deploying local docker cluster

## Prerequisites

* [Docker Engine](https://docs.docker.com/engine/install/) - Minimum version 20.10.23
* [Docker compose v.1.x](https://docs.docker.com/compose/install/) - Minimum version 1.29

### `mbft` consensus

When deploying with `mbft` consensus, there are some additional dependencies:

* [go 1.20.x](https://go.dev/dl/)

## Local development

Running `metachain` local cluster with docker can be done very easily by using provided `scripts` folder
or by running `docker-compose` manually.

### Using provided `scripts` folder

***All commands need to be run from the repo root / root folder.***

* `scripts/cluster metabft --docker` - deploy environment with `metabft` consensus
* `scripts/cluster mbft --docker` - deploy environment with `mbft` consensus
* `scripts/cluster {metabft or mbft} --docker stop` - stop containers
* `scripts/cluster {metabft or mbft}--docker destroy` - destroy environment (delete containers and volumes)

### Using `docker-compose`

***All commands need to be run from the repo root / root folder.***

#### use `metabft` PoA consensus

* `export CHAIN_CONSENSUS=metabft` - set `metabft` consensus
* `docker-compose -f ./docker/local/docker-compose.yml up -d --build` - deploy environment

#### use `mbft` consensus

* `export CHAIN_CONSENSUS=mbft` - set `mbft` consensus
* `docker-compose -f ./docker/local/docker-compose.yml up -d --build` - deploy environment

#### stop / destroy

* `docker-compose -f ./docker/local/docker-compose.yml stop` - stop containers
* `docker-compose -f ./docker/local/docker-compose.yml down -v` - destroy environment

## Customization

Use `docker/local/metachain.sh` script to customize chain parameters.
All parameters can be defined at the very beginning of the script, in the `CHAIN_CUSTOM_OPTIONS` variable.
It already has some default parameters, which can be easily modified.
These are the `genesis` parameters from the official [docs](https://wiki.Metachain.technology/docs/chain/operate/param-reference/).  

Primarily, the `--premine` parameter needs to be edited to include the accounts that the user has access to.

## Considerations

### Build times

When building containers for the first time (or after purging docker build cache),
it might take a while to complete, depending on the hardware that the build operation is running on.

### Production

This is **NOT** a production ready deployment. It is to be used in *development* / *test* environments only.
For production usage, please check out the official [docs](https://wiki.Metachain.technology/docs/chain/).
