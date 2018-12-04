
# multinodetester 编译说明
    1. 切换到当前目录
    2. 运行  go build 后，当前目录会生成multinodetester可执行文件

# multinodetester 功能说明
    multinodetester 可以在本地创建多个cosd的data目录，方便大家进行P2P和共识机制的单机调试
    生成的文件放在 ~/.coschain/testcosd_XXX 下面
    如果有配置冲突的情况，可以自行修改代码
    
### init 
    创造3个数据目录，以供后面运行多个进程
    ./multinodetester init 3   
    
### clear 
    清除所有数据目录
    ./multinodetester clear 
    
### stop
    杀掉所有cosd的进程 (其实就是：pkill -9 cosd)
    ./multinodetester stop 
    
### start
    启动3个cosd的进程，需要指定cosd的全路径，启动后所有的输入输出流会在命令行中
     ./multinodetester start /Users/yykingking/go/src/github.com/coschain/contentos-go/cmd/cosd/cosd 3
    