本文档将向您展示与出块节点相关的操作流程，包括：如何搭建节点、如何升级节点、节点如何退出、如何查看节点当前状态等


## 如何搭建一个出块节点

如果你想要自己运行起来的出块节点参与主网出块，可以参考下面的步骤

### 前提

确保节点的 20338、8888、8080 三个端口可以被公网访问到

### 1.创建Contentos账号

可以使用 [网页钱包](https://wallet.contentos.io/) 来创建一个账号，拿到自己的用户名、公钥、私钥。

注意：保证账号中至少包含30000个VEST。如果你还没有VEST，那么你可以通过以下操作步骤获取

```
1. 首先在交易所买到 BEP2 格式 COS
2. 通过 [COS币映射程序](http://swapcos.contentos.io) 将BEP2-COS转换为主网币
3. 然后通过[网页钱包](https://wallet.contentos.io/) 将COS转换为VEST。
```

### 2.编译节点程序

保证代码处在`master`分支，我们将该分支作为稳定分支供外部使用，具体的步骤参考
[编译流程](https://github.com/coschain/contentos-go#building-the-source)
[初始化流程](https://github.com/coschain/contentos-go#initialization)

### 3.修改配置文件、启动节点、注册账号成为block producer

#### 修改配置文件

在config.toml 修改以下内容:
```
  BootStrap = false (注意，这个值必须设置为false)
  LocalBpName = your_account_name (刚才第一步创建的账号名)
  LocalBpPrivateKey = your_private_key (刚才第一步创建账号生成的私钥)
  SeedList = ["3.210.182.21:20338","34.206.144.13:20338"] (主网的种子节点，需要正确配置才能保证连上网络中其他节点)
```

#### 启动节点等待数据同步完成
我们提供了对主网数据的快速同步功能，通过下面的命令可以开启此功能
```
./cosd fast-sync

```
上述指令只是将主网数据下载下来，并放置在你的配置文件中`DataDir`指定的路径下，并未将节点启动，且此时你的节点数据仍然落后于主网。
你可以执行下面的命令将cosd进程启动并且自动去同步余下的数据（你也可以跳过上述指令，直接运行下面的指令，但这可能导致同步过程花费很长的时间）

```
./cosd start

```

注意：同步过程需要花费一定的时间，如果数据没有同步完成，那么钱包的任何发起交易操作都会返回失败，失败信息如下

```

rpc error: code = Unknown desc = consensus not ready
	
```

#### 使用钱包将账号注册为producer

当节点数据同步完成后，你可以执行以下命令将自己注册为block producer
```
  ./wallet-cli
  import your_account_name privateKey_of_your_account
  bp register your_account_name publicKey_of_your_account
```
其中 账号名， 公钥，私钥 等参数在第一步创建账号的时候均可以获取到

### 4.拉选票
操作成功后，你可以在 [BP列表](https://explorer.contentos.io/#/bp/) 中看到自己的名字了。 因为系统只会让排名前21的BP来出块获取奖励，所以接下来你要做的事情就是保证自己的选票足够多，要么去呼吁其他人[投票给你](https://wallet.contentos.io/#/bpvote)，要么质押更多的VEST

## 如何升级节点

当有新的代码版本发布的时候，就需要进行节点升级。如果你已经将自己注册为producer，那么你就有可能参与出块，在进行节点升级的时候需要格外注意

### 1.解锁账户

编译`wallet-cli`工具，具体的编译流程可以参考[编译流程](https://github.com/coschain/contentos-go#building-the-source)

编译完成后执行 `./wallet-cli` 将工具运行，首先将自己的账户导入到`wallet-cli` 中
```
import YourAccountName YourPrivateKey
```
执行完成后设置密码，以后每次对账户进行解锁操作，都需要输入这个密码，第一次导入之后的账户默认处于解锁状态。如果你之前已经将账户导入到了`wallet-cli` 中，那么可以直接执行
```
unlock YourAccountName
```
并输入之前在导入时设置的密码，可以通过
```
list
```
指令查看之前是否已经导入过账户

### 2.退出出块节点

解锁账户后，需要使自己的账户不再参与出块，此过程需要执行以下指令
```
bp enable YourAccountName --cancel
```
执行完成后，请确认自己的名字已经从 [BP列表](https://explorer.contentos.io/#/bp/)中移除

### 3.等待至少2分钟

这一步非常重要，因为即使上一步的指令执行成功了，但是在当前的出块周期内，你还是有可能参与出块，如果此时立即将进程停掉，还是有可能出现丢块的情况

### 4.拉取代码、编译并运行

从远端拉取最新代码，编译并运行，等待自己的节点同步完成，如果你对这些过程还不熟悉，请参考[如何搭建一个出块节点](https://github.com/coschain/contentos-go/blob/master/bp_cn.md#如何搭建一个出块节点)

### 5.重新参与到出块过程

此过程依然需要通过钱包进行操作，所以要保证自己的账户处于解锁状态，然后执行
```
bp enable YourAccountName
```
执行成功后，就可以在 [BP列表](https://explorer.contentos.io/#/bp/) 中重新看到自己的名字了

## 节点如何退出

此过程和升级节点的步骤差不多，都需要先保证自己的节点不再参与出块，然后才能执行后续的步骤，其中，前三步是相同的

### 1.解锁账户

执行步骤与**如何升级节点**中的步骤相同

### 2.退出出块节点

执行步骤与**如何升级节点**中的步骤相同

### 3.等待至少2分钟

执行步骤与**如何升级节点**中的步骤相同

### 4.后续操作

此时，你才可以按照自己的意愿对节点进行操作

## 如何查看节点当前状态

### 1.编译并运行钱包

钱包的编译可以参考[编译流程](https://github.com/coschain/contentos-go#building-the-source)，然后执行 `./wallet-cli` 将其运行

### 2.查看节点状态

钱包启动后，可以通过
```
chainstate
```
指令查看节点当前的状态，以下只列出返回结果中较重要的一些信息
```
GetChainState detail: {
	"state": {
		"last_irreversible_block_number": 3718245,（不可逆块的高度）
		"last_irreversible_block_time": 1573115290,
		"dgpo": {
			"head_block_id": "65bc380000000000193ae8e7ee78016ffd9fb708a4e551a8ebb35e197de7ebbf",
			"head_block_number": 3718245,（链的最大高度）
			"current_block_producer": "contentosbp2",（链的最大高度的块是由哪个节点生产的）
		}
	}
}

```