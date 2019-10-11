# 如何搭建一个出块节点

如果你想要自己运行起来的出块节点参与主网出块，可以参考下面的步骤

## 1.创建Contentos账号

可以使用[网页钱包](https://wallet.contentos.io/)来创建一个账号，拿到自己的用户名公钥私钥。

注意：保证账号中至少包含30000个VEST。如果你还没有VEST，那么你可以通过以下操作步骤获取

 1. 首先在交易所买到`BEP2`格式的COS
 2. 通过[COS币映射程序](http://swapcos.contentos.io)将`BEP2-COS`转换为主网币
 3. 然后通过[网页钱包](https://wallet.contentos.io/)将COS转换为VEST

## 2.编译节点程序

切换到分支`origin/release-v1.0.0`，具体的步骤参考

```sh
git clone git@github.com:coschain/contentos-go.git
cd contentos-go
git checkout -b origin/release-v1.0.0
git pull origin release-v1.0.0

cd cmd/cosd
go build
./cosd init

cd ../wallet-cli/
go build
```

## 3.修改配置文件、启动节点、注册账号成为block producer

### 修改配置文件

```
cd contentos-go
vim config.toml
```

编辑以下内容:

```
[Consensus]
BootStrap = false (注意，这个值必须设置为false)
LocalBpName = your_account_name (刚才第一步创建的账号名)
LocalBpPrivateKey = your_private_key (刚才第一步创建账号生成的私钥)

[P2P.Genesis]
SeedList = ["3.210.182.21:20338","34.206.144.13:20338"] (主网的种子节点，需要正确配置才能保证连上网络中其他节点)
```

### 启动节点等待数据同步完成

使用下面的命令将cosd进程启动并且自动去同步数据

```
cmd/cosd/cosd start
```

***注意：同步过程可能需要等待很久，如果数据没有同步完成，那么钱包的任何发起交易操作都会返回失败，失败信息如下***

```
rpc error: code = Unknown desc = consensus not ready
```

### 使用钱包将账号注册为producer

当节点数据同步完成后，你可以执行以下命令将自己注册为block producer

```
./wallet-cli
import your_account_name privateKey_of_your_account
bp register your_account_name publicKey_of_your_account
```

其中`your_account_name`，`privateKey_of_your_account`，`publicKey_of_your_account`等参数在第一步创建账号的时候均可以获取到

## 4.拉选票

操作成功后，你可以在[BP列表](https://explorer.contentos.io/#/bp/)中看到自己的名字了。因为系统只会让排名前21的BP来出块获取奖励，所以接下来你要做的事情就是保证自己的选票足够多，要么去呼吁其他人[投票给你](https://wallet.contentos.io/#/bpvote)，要么质押更多的VEST。
