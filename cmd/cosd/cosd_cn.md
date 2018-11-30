# cosd 编译说明
    1. 需要 go version >= 1.11 
    2. 切换到当前目录
    3. 运行  go build 后，当前目录会生成cosd可执行文件

# cosd 使用说明


### 1. 初始化配置文件

    ./cosd init
    
    此命令会在 ~/.coschain/ 目录下创建配置文件，默认生成initminer账号的公私钥，只需运行一次即可
    
            
### 2. 启动节点

    ./cosd start
    
    默认配置下3秒中之后开始启动单节点打包，会有如下log打印出来证明打包开始
    
    #### [2018-11-30 20:13:23] DEBUG DPoS: generated block: <num 1> <ts 1543580003> file=dpos.go func=consensus.(*DPoS).start line=227 pid=17774
    #### [2018-11-30 20:13:23] DEBUG pushBlock #1 file=dpos.go func=consensus.(*DPoS).pushBlock line=386 pid=17774
    #### [2018-11-30 20:13:23] DEBUG ### saveReversion, num:1 rev:5 file=controller.go func=app.(*Controller).saveReversion line=923 pid=17774
    #### [2018-11-30 20:13:23] DEBUG DPoS shuffle: active producers: [initminer] file=dpos.go func=consensus.(*DPoS).pushBlock line=424 pid=17774


### 3. 启动本地wallet和节点进行交互
    参考  https://github.com/coschain/contentos-go/blob/master/cmd/wallet-cli/wallet_doc_cn.md
