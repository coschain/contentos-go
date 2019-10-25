This tool helps you set up a Contentos public chain mainnet node step by step through command line interaction.
The whole process mainly consists of two parts: generating a config file and starting the node. Here are the specific steps

## Precondition

Install go environment and it requires [Go 1.11.4+](https://golang.org/dl/)

Make sure all running cosd processes have been stopped
```
pkill cosd
```

Then execute the following commands under the current path in order
```
go build
./autosetup
```

## Step 1. Set node name

Set the name of your node, if you want to use default node name, please enter `d`
```
Enter your node name (If you want to use default name, enter d)
```

If the node name already has a config file, it will display

```
Already has a config file, delete and init a new one? (yes/no)
```

You can choose to use the old config file or create a new one

## Step 2. Set block producer name
First, you will be asked if you want to start a block producer node.
```
Do you want to start a bp node? (yes/no)
```
If you don't want to start a block producer node, please enter `no` and skip step 2 and 3. If you want, please enter your account name (You can use our [online wallet](https://wallet.contentos.io/) to create your Contentos mainnet account)
```
Enter your account name:
```

## Step 3. Set block producer private key
```
Enter your private key:
```

## Step 4. Set seed node

Set the seed node information

If you want to connect a chain, you should connect the seed node first.
```
Enter seed node list: (e.g. ip1:port1,ip2:port2)
```

* mainnet seed node : 3.210.182.21:20338,34.206.144.13:20338

These two nodes are our official seed nodes. Of course you can also use the IP of any known mainnet node.

## Step 5. Set log level

Set your log level, if you want to use default log level, please enter `d`, default log level is `debug`
```
Enter your log level ( debug / info / warn / error / fatal / panic, use default level enter d)
```

## Step 6. Set data directory

Set your data directory, if you want to use default data directory, please enter `d`, default data directory is `$HOME/.coschain/(your node name)`.
Data directory is mainly used to store the database of the public chain. If you don't want to use default path, we recommend that you enter an absolute path.
```
Enter your data directory, use default directory enter d:
```

## Step 7. Start your node

If you don't want to start your node, please skip step 7 and 8
```
Do you want to start the node right now? (yes/no)
```

## Step 8. Clear local data

```
Clear local data? (yes/no)
```
If you want to clear data in your data directory, please choose `yes`, otherwise choose `no`


Now, you just start the node and run it in the background. If you want your node to produce blocks, firstly, you should sync all data from the chain;
secondly, ensure that your account has a vest balance of at least 30,000, finally, register your account as a block producer and call for vote. For detail information, you can refer [this](https://github.com/coschain/contentos-go/blob/dev/bp.md)