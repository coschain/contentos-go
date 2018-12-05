# multinodetester2 编译说明
```
1. 切换到当前目录
2. 运行  go build 后，当前目录会生成multinodetester2可执行文件
```

# multinodetester2 功能说明
```
multinodetester2 可以在本地创建多个cosd的data目录，方便大家进行P2P和共识机制的单机调试
生成的文件放在 ~/.coschain/testcosd_XXX 下面
如果有配置冲突的情况，可以自行修改代码
```### init
```
创造3个数据目录，以供后面在单进程中模拟多个节点（默认初始化3个）
./multinodetester2 init 3   
```### clear
```
清除所有数据目录
./multinodetester2 clear 
```### start
```
模拟3个cosd的节点（默认模拟3个）
 ./multinodetester2 start -n 3
```