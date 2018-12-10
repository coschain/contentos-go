
# Contentos-go

official golang impementation of the Contentos protocol

Contentos Website: https://www.contentos.io

Contentos White Paper: https://www.contentos.io/subject/home/pdfs/white_paper_en.pdf

Follow us on https://twitter.com/contentosio

Join discussion at https://t.me/ContentoOfficialGroup

**WARNING:** For now, the branch is under active developing. Thus mostly it stabilized, but we are still introducing some breaking changes.

**Note*:* Requires [Go 1.11+](https://golang.org/dl/)

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

the contento-go composed with two executables as above.

**cosd**: the daemon to run a local blockchain

**wallet**: the cli to interactive with chain.

## Running cosd

### Initializing

```bash
cosd init
```

it will use *cosd* as default node name. If you prefer your own name, using:

```bash
cosd init -n yourownname
```

### Configuration

After being initialized, configurations will be found in homedir/.cosd/nodename

The nodename is cosd or yourownname.

You can modify it if you like as you know actually what you are doing.


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

Thank you for considering to help out with the source code! We welcome contributions from anyone on the internet, and are grateful for even the smallest of fixes!

If you'd like to contribute to contento-go, please fork, fix, commit and send a pull request for the maintainers to review and merge into the main code base. Or you can contact us directly by join telegram.
