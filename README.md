# Contentos-go
[![Build Status](https://travis-ci.com/coschain/contentos-go.svg?branch=master)](https://travis-ci.com/coschain/contentos-go)
[![Code Coverage](https://codecov.io/gh/coschain/contentos-go/branch/master/graph/badge.svg)](https://codecov.io/gh/coschain/contentos-go)

official golang impementation of the Contentos protocol

Contentos Website: https://www.contentos.io

Contentos White Paper: https://www.contentos.io/subject/home/pdfs/white_paper_en.pdf

Follow us on https://twitter.com/contentosio

Join discussion at https://t.me/ContentoOfficialGroup

**WARNING:** The branch is under heavy development. Breaking changes are actively added.

**Note**: Requires [Go 1.11.4+](https://golang.org/dl/)

## Building the source

```bash
make build_cosd
```

And

```bash
make build_wallet
```

Or, to build both

```bash
make build
```

if prefer to install into /bin:

```bash
make install_cosd
```

And

```bash
make install_wallet
```

Or

```bash
make install
```

## Testing the source

```bash
make test
```

## Executables
The contento-go contains two executables as follow:

**cosd**: the daemon to run a local blockchain

**wallet**: the cli to interactive with chain.

## Run in docker

### Build the image from source with docker

Move to the root directory of source code and run the following command:

```bash
docker build -t=contentos .
```

Don’t forget the dot at the end of the line, it indicates the build target is in the current directory.

When the build process is over you can see a message indicating that it is ‘successfully built’.

### Run the container

The below command will start the container as a daemonized instance. When the container is started, cosd started simultaneously.

```bash
docker run -d --name contentosd-exchange -p 8888:8888 -p 20338:20338 -v /path/to/coschain:/root/.coschain contentos

```

The `--name` flag assigns a name to the container, and the `-v` flag indicates how you map directories outside of the container to the inside, the path before the `:` is the directory on your disk.`-p` flag publishes a container’s port to the host

You can see the running container by using the command  `docker ps`.

To follow along with the logs, use `docker logs -f contentosd-exchange`.

### Run the wallet-cli

The following command will run the wallet-cli from inside the running container:

```bash
docker exec -it contentosd-exchange /usr/local/src/contentos-go/bin/wallet-cli

```

## Running cosd

### Initialization

```bash
cosd init
```

cosd is adopted in default as the node name. To change it, use:

```bash
cosd init -n yourownname
```

### Configuration
After initialization, configurations will be found under homedir_.cosd_nodename

The nodename is cosd or yourownname.

You can modify it if you like as long as you know what you are doing.

## Running

```bash
cosd start
```

if you have named your node, using:

```bash
cosd start -n yourownname
```

### Interaction
enter

```bash
wallet
```

to get into interactive mode.

You can using some commands as below:

* account
* bp
* claim
* close
* create
* follow
* genKeyPair
* import
* info
* locked
* list
* load
* lock
* post
* reply
* transfer
* unlock
* transfer_vesting
* vote

you can add `--help` or `help [command]` to get more detail infos.

## Contribution
Contributions are welcomed.

If you'd like to help out with the source code, please send a pull request. Or you can contact us directly by joining telegram.