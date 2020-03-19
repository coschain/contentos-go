This document will show you the specific upgrade steps for the third hard fork of the Contentos mainnet.

Here are the specific upgrade steps, if your node is not a block producer, you can skip step 1,2 and 7.

## Step 1. Disable your account as a block producer
If your node is a block producer, before upgrading, you should ensure that your node is no longer a block producer node.
To achieve this goal, you can use `wallet-cli`.

### Firstly start the tool
```
  cd /path/to/your/contentos-go/project
  cd cmd/wallet-cli/
  ./wallet-cli
```

### Secondly `import` or `unlock` your account
If you have used the `wallet-cli` before and have imported your account, you can execute this command
```
  unlock your_account_name
```
and enter your wallet password.

If you haven't used `wallet-cli` before, you can execute this command to import your account
```
  import your_account_name privateKey_of_your_account
```
and set your wallet password.

### Lastly disable your account as a block producer
Execute the following command to disable your account as a block producer
```
  bp enable your_account_name --cancel
```

## Step 2. Wait at least 2 minutes
**This step is very important**, because even if the command of the previous step is executed successfully, 
you may still participate to produce block during the current period. If you stop the process immediately, some blocks will be lost.

## Step 3. Stop the `cosd` process
```
  pkill cosd
```

## Step 4. Pull the latest code and compile `cosd`
You should pull the latest code on `master` branch
```
  cd /path/to/your/contentos-go/project
  git checkout master
  git pull
```
and compile `cosd`
```
  cd cmd/cosd/
  go build
```

## Step 5. Use `fast-sync` to fetch the new main network data
We provide a fast synchronization function for the main network data. 
This function can be enabled by the following command
```
  ./cosd fast-sync

```

## Step 6. Start the `cosd` and wait to sync to the latest state
The previous step just download the mainnet data and place it in the path specified by `DataDir` in your config file.
You can use the following command to run the `cosd` as a backstage process
```
  nohup /path/to/cosd/executable/file start 2> /path/to/std/error/log 1>/dev/null &
```
Now `cosd` is already running as a backstage process but your node data is still fall behind the mainnet at this moment.
You need to wait for your node to sync to the latest state, this will take some time.

You can observe your node status through our [blockchain browser](http://explorer.contentos.io/#/)
and the following image shows you how to get the browser to connect to your own node
![browser](doc/technical-whitepaper/assets/5.png)

**Be careful, just enter your IP and do not change the port(8080), use http and do not use https**

When the `Confirmation delay Time` shows 0, 1 or 2 sec ago, it means that your node has already sync to the latest state

## Step 7. Enable your account as a block producer
In step 1 you disabled your account as a block producer and now you need to enable it back.
So you need `wallet-cli` again.
Start the `wallet-cli` and `unlock` your account, if you forget hw to do it, please refer to step 1,
then execute the following command to enable your account as a block producer
```
  bp enable your_account_name
```

At this point, you have completed all the upgrade steps
