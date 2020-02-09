This document will show you the operational procedures related to the Block Producer node,
including how to set up the node, how to upgrade the node, how to exit the node, how to view the current state of the node, etc.

## How to set up a Contentos Block Producer node

If you want to set up a Contentos Block Producer node, please follow the steps below.

### Precondition

Ensure that the three ports of the node 20338, 8888, and 8080 can be accessed by the public network.

### 1.Create your Contentos account

Please use our [online wallet](https://wallet.contentos.io/) to create your Contentos account and
get your username, public key, private key.

Note: The account must contain at least 30,000 VEST.
If you don't, please follow the steps below

```
1. buy BEP2 format COS on the exchange
2. Convert BEP2-COS to real COS via [COS Coin Mapper] (http://swapcos.contentos.io)
3. convert COS to VEST via [Web Wallet] (https://wallet.contentos.io/).
```

### 2.Build and Initialization

Ensure that the code is in the `master` branch, we use this as a stable branch for external use.

pull source code
```
git clone git@github.com:coschain/contentos-go.git
```
build cosd
```
cd cmd/cosd
go build
```
Initialization

init will create a folder to hold cosd's running data,this will create a folder `$HOME/.coschain/cosd`
```
./cosd init
```

### 3.Modify your config file、start the node and register your account as a block producer

#### Modify config file

Please modify the following things in the file config.toml:
(The directory of config.toml is `$HOME/.coschain/cosd`)
```
  BootStrap : false (Be careful, this must be set to false)
  LocalBpName : your account name
  LocalBpPrivateKey : private key of your Contentos account
  SeedList : ["3.210.182.21:20338","34.206.144.13:20338"] (Set this to the seed nodes of contentos main net)
```

#### Start the node and register your account as a block producer

We provide a fast synchronization function for the main network data. This function can be enabled by the following command
```
./cosd fast-sync

```
The above command just download the mainnet data and place it in the path specified by `DataDir` in your config file.
The node is not started, and your node data is still behind the mainnet at this moment.
You can execute the following command to start the cosd process and automatically synchronize the remaining data
(You can also skip the above command and run the following directly, but this may cause the synchronization process to take a long time)
```
  ./cosd start
```

Note: The synchronization process will take some time. 
If the data is not synchronized, then any transaction initiated by the wallet will fail. 
The failure message is as follows:

```

rpc error: code = Unknown desc = consensus not ready
	
```

When everything is done, please use our wallet to register your account as a block producer,
you should execute the following commands in order
```
  ./wallet-cli
  import your_account_name privateKey_of_your_account
  bp register your_account_name publicKey_of_your_account
```

### 4.Block Producer Vote
Now you have become a Block Producer, you should call for more people to vote for you, the more people who vote for you, the more votes you have.
We will sort by number of votes in descending order, only the first 21 Block Producer can generate block. Here is web page for [BP Vote](https://wallet.contentos.io/#/bpvote)

## How to upgrade the node

When a new version of the code is released, node upgrade is required. 
If you have already registered yourself as a producer, you are likely to participate to produce block, and you need to pay special attention when doing node upgrades.

### 1.Unlock account

Compile `wallet-cli`
```
cd contentos-go/cmd/wallet-cli/
go build
```

After compilation is complete, execute `./wallet-cli`. First import your account into `wallet-cli`
```
import YourAccountName YourPrivateKey
```

After the execution is completed, set the password. You will need to enter this password each time you unlock your account.
The account after the first import is unlocked by default. If you have already imported your account into `wallet-cli`, 
you can execute the following command directly.
```
unlock YourAccountName
```
enter your password, you can use
```
list
```
to check if you have already imported an account before

### 2.Exit the node
After unlocking the account, you need to make your account no longer participate to produce block. 
This process requires the following command.
```
bp enable YourAccountName --cancel
```
After the execution is complete, please confirm that your name has been removed from [BP List](https://explorer.contentos.io/#/bp/)

### 3.Wait at least 2 minutes

This step is very important, because even if the command of the previous step is executed successfully, 
you may still participate to produce block during the current period. If you stop the process immediately, some blocks will be lost.

### 4.Get code, compile and run

Get the latest code from remote, compile and run, wait for your node to complete synchronization. 

pull latest code
```
git pull
```
build and run cosd
```
cd cmd/cosd
go build
./cosd start
```

### 5.Re-engagement

This process still needs to be operated through the wallet, so make sure your account is unlocked and then execute
```
bp enable YourAccountName
```
After successful execution, you can see your name again in [BP List](https://explorer.contentos.io/#/bp/)

## How to exit the node

This process is similar to the steps to upgrade a node. 
You need to ensure that your node is no longer to participate to produce block before you can perform subsequent steps.
Among them, the first three steps are the same

### 1.Unlock account

The steps are the same as those in **How to upgrade the node**

### 2.Exit the node

The steps are the same as those in **How to upgrade the node**

### 3.Wait at least 2 minutes

The steps are the same as those in **How to upgrade the node**

### 4.Follow-up operation

At this point, you can operate the node as you wish.

## How to view the current state of the node

### 1.Compile and run the wallet

build wallet
```
cd contentos-go/cmd/wallet-cli/
go build
```

Then execute `./wallet-cli` to run it

### 2.View node status

After the wallet is started, you can execute
```
chainstate
```
to view the current status of the node. 
The following only lists some of the more important information in the returned results.
```
GetChainState detail: {
	"state": {
		"last_irreversible_block_number": 3718245,
		"last_irreversible_block_time": 1573115290,
		"dgpo": {
			"head_block_id": "65bc380000000000193ae8e7ee78016ffd9fb708a4e551a8ebb35e197de7ebbf",
			"head_block_number": 3718245,
			"current_block_producer": "contentosbp2",（who produce the head block）
		}
	}
}

```