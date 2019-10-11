# How to set up a Contentos Block Producer node

If you want to set up a Contentos Block Producer node, please follow the steps below.

## Precondition

Ensure that the three ports of the node 20338, 8888, and 8080 can be accessed by the public network.

## 1.Create your Contentos account

Please use our [online wallet](https://wallet.contentos.io/) to create your Contentos account

## 2.Build and Initialization

Switch to branch `origin/release-v1.0.0`.To acquire detail instruction, you can refer
[build instruction](https://github.com/coschain/contentos-go#building-the-source)
[Initialization](https://github.com/coschain/contentos-go#initialization)

## 3.Modify your config file、start the node and register your account as a block producer

### Modify config file

Please modify the following things in the file config.toml:
```
  BootStrap : false (Be careful, this must be set to false)
  LocalBpName : your account name
  LocalBpPrivateKey : private key of your Contentos account
  SeedList : ["3.210.182.21:20338","34.206.144.13:20338"] (Set this to the seed nodes of contentos main net)
```

### Start the node and register your account as a block producer

The following command can run the node and wallet
```
  ./cosd start
  ./wallet-cli
```

When the node is start up, please use our wallet to register your account as a block producer,
you should execute the following commands in order
```
  ./wallet-cli
  import your_account_name privateKey_of_your_account
  bp register your_account_name publicKey_of_your_account
```

## 4.Block Producer Vote
Now you have become a Block Producer, you should call for more people to vote for you, the more people who vote for you, the more votes you have.
We will sort by number of votes in descending order, only the first 21 Block Producer can generate block. Here is web page for BP Vote(https://wallet.contentos.io/#/bpvote)