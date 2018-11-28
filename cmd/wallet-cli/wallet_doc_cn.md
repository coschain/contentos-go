# wallet-cli 编译说明
    1. 切换到当前目录
    2. 运行  go build 后，当前目录会生成wallet-cli可执行文件

# wallet-cli 使用说明

### 首先启动cosd，然后运行wallet-cli后进入命令行交互

### 创建账号

    //首先需要引入一个已经存在的账号
    import initminer 27Pah3aJ8XbaQxgU1jxmYdUzWaBbBbbxLbZ9whSH9Zc8GbPMhw
            // 此处需要输入密码，后面如有需要使用 unlock initminer 来解锁账号
            ====> enterpassword
    
    //创建账号
    create initminer yyking
            // 此处需要输入密码，后面如有需要使用 unlock yyking 来解锁账号
            ====> enterpassword
            
    //查询账号信息
    account get yyking
        会看到有如下输出：
        GetAccountByName detail: {"account_name":{"value":"yyking"},"coin":{},"vest":{"value":1},"created_time":{"utc_seconds":1543375443}}
        
### 转账

    //首先需要解锁一个已经存在的账号
    unlock initminer
            // 此处需要输入之前设置的密码
            ====> enterpassword
    
    //转账
    transfer initminer yyking 99
                      
    //查询账号信息
    account get yyking
        //会看到有如下输出， coin为99 ：
        GetAccountByName detail: {"account_name":{"value":"yyking"},"coin":{"value":99},"vest":{"value":1},"created_time":{"utc_seconds":1543375443}}
    
### 投选票给BP节点

    //首先生成一个新的公私钥对
    genKeyPair
    // 输出===========>
           Public  Key:  COS7Z7oHh2NHGjmQqZULNJZch9rNZfztzfg5AmSHQgozzGagBokm5
           Private Key:  2gHCwYnNBrgij6TRJoyLTSZDWjJWzSQLby8GJvHBZjTuzvLc2X
 
    
    //注册为BP节点
    bp register yyking COS7Z7oHh2NHGjmQqZULNJZch9rNZfztzfg5AmSHQgozzGagBokm5            // 此处需要输入密码，后面如有需要使用 unlock yyking 来解锁账号
            
    //设置initminer投票给yyking
    bp vote initminer yyking
    
    //取消设置initminer投票给yyking
    bp vote initminer yyking --cancel
    
### 查看帮助
 
可以通过

    help 
    help create
    help post
    
来查看常见命令的使用方法


