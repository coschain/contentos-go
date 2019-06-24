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

docker run -d --name contentosd-exchange -p 8888:8888 -p 20338:20338 -v /path/to/blockchain:/root/.coschain -v /path/to/project/home/directory/config.toml:/root/.coschain/cosd/config.toml contentos

```

The `--name` flag assigns a name to the container, and the `-v` flag indicates how you map directories outside of the container to the inside, the path before the `:` is the directory on your disk.`-p` flag publishes a container’s port to the host.

If you want to run the node as a block producer,please modify the following things in the file config.toml:

```bash

  LocalBpName : your account name
  LocalBpPrivateKey : private key of your account

```

The content in the home directory file set.txt indicates whether you delete local blockchain when the container start up.If you don't want to delete it, please change `delete` to `reserve`.

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
## Running multi nodes in single machine

if you want to set up a contentos network in single machine, you need to do:

1.run cosd init -n name to create different node folders,then edit all config.toml,make sure HTTPListen and RPCListen and NodeConsensusPort and NodePort and HealthCheck all unique to other node's config.toml.
```
./cosd init -n cos1
./cosd init -n cos2
./cosd init -n cos3
./cosd init -n cos4
vi ~/.coschain/cos1/config.toml
vi ~/.coschain/cos2/config.toml
vi ~/.coschain/cos3/config.toml
vi ~/.coschain/cos4/config.toml
```

2.start first cosd,use wallet connect cosd via RPC address according to config.toml.
```
./cosd start -n cos1
./wallet-cli
switchport ip_to_your_node:8888
import initminer privateKey_of_initminer
```

3.create 3 new accounts and use info command to get thier public keys and priviate keys.
```
stake initminer initminer 1.000000
create initminer witness1
create initminer witness2
create initminer witness3
info witness1
info witness2
info witness3
```

4.edit other node's config.toml,change BootStrap to false,set LocalBpName and LocalBpPrivateKey that you
just created in step 3.

5.start remain cosd.
```
./cosd start -n cos2
./cosd start -n cos3
./cosd start -n cos4
```

6.use wallet again to regist 3 accounts as new bp
```
unlock iniminer
bp register witness1 witness1_publicKey
bp register witness2 witness2_publicKey
bp register witness3 witness3_publicKey
```

now job is finished.

optinal, the first cosd has been changed to a observe node since you regist bp, if you want first cosd also become a bp node, you need to create a new account,modify first cosd's config.toml,change BootStrap to false,set LocalBpName and LocalBpPrivateKey to new account's info(same as step 4),restart first cosd, regist new account as bp(same as step 6).

## Running multi nodes in multi machines

assume you have 4 machines and want to set up a contentos network, here the steps
you need to do:

1.choice a machine,run cosd init then edit config.toml,modify all ip relative items from 127.0.0.1 to
your custom.
```
./cosd init
vi ~/.coschain/cosd/config.toml
```

2.start first cosd,use wallet connect cosd via RPC address according to config.toml.
```
switchport ip_to_your_node:8888
import initminer privateKey_of_initminer
```

3.create 3 new accounts and remember thier public keys and private keys.
```
stake initminer initminer 1.000000
create initminer witness1
create initminer witness2
create initminer witness3
info witness1
info witness2
info witness3
```

4.do cosd init on other 3 machines,edit config.toml,change BootStrap to false,set LocalBpName and LocalBpPrivateKey that you
just created in step 3.

5.start remain cosd on each machine.
```
./cosd start 
```

6.use wallet again to regist 3 accounts as new bp
```
unlock iniminer
bp register witness1 witness1_publicKey
bp register witness2 witness2_publicKey
bp register witness3 witness3_publicKey
```

now job is finished.

optinal, the first cosd has been changed to a observe node since you regist bp, if you want first cosd also become a bp node, you need to create a new account,modify first cosd's config.toml,change BootStrap to false,set LocalBpName and LocalBpPrivateKey to new account's info(same as step 4),restart first cosd, regist new account as bp(same as step 6).

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
