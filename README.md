
# Contentos-go

[![Build Status](https://travis-ci.com/coschain/contentos-go.svg?branch=master)](https://travis-ci.com/coschain/contentos-go)
[![Code Coverage](https://codecov.io/gh/coschain/contentos-go/branch/master/graph/badge.svg)](https://codecov.io/gh/coschain/contentos-go)

official golang impementation of the Contentos protocol

Contentos Website: https://www.contentos.io

Contentos White Paper: https://www.contentos.io/subject/home/pdfs/white_paper_en.pdf

Follow us on https://twitter.com/contentosio

Join discussion at https://t.me/ContentoOfficialGroup

**WARNING:** The branch is under heavy development. Breaking changes are actively added.

**Note*:* Requires [Go 1.11.4+](https://golang.org/dl/)

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

## Running cosd

### Initialization

```bash
cosd init
```

*cosd* is adopted in default as the node name. To change it, use:

```bash
cosd init -n yourownname
```

### Configuration

After initialization, configurations will be found under homedir/.cosd/nodename

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
