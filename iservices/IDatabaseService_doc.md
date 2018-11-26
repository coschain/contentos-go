# IDatabaseService 接口说明

IDatabaseService是coschain的存储服务。

它提供key-value store的HAS/GET/PUT/DELETE基本操作，原子性批量写，遍历，嵌套交易和数据回滚的功能。



## 基本操作: HAS/GET/PUT/DELETE

所有基本操作都是线程安全的。

### Has(key []byte) (bool, error)

Has()返回存储中是否存在指定的key。如果存在，返回(true, nil)；不存在，返回(false, nil)。查询出错返回(false, error)。

### Get(key []byte) ([]byte, error)

Get()返回存储中指定的key对应的value。如果key存在，返回(value, nil)，key不存在或其他错误，返回(nil, error)。

### Put(key []byte, value []byte) error

Put()向存储中添加新的key-value对，如果key已经存在，更新它的value。成功返回nil，失败返回error。

### Delete([]key) error

Delete()从存储中删除指定的key。成功返回nil，失败返回error。

如果指定的key不存在，Delete()也是成功的，即Delete()只要达到“存储中不存在指定key”的结果即为成功。当数据库操作失败或发生其他逻辑错误时，才会返回error。



## 原子性批量写：IDatabaseBatch

IDatabaseBatch用于原子地执行多个写操作（Put或Delete）。IDatabaseService可以创建、销毁IDatabaseBatch。

### NewBatch() IDatabaseBatch

NewBatch()创建一个IDatabaseBatch。

### DeleteBatch(b IDatabaseBatch)

DeleteBatch()销毁一个IDatabaseBatch。

### IDatabaseBatch接口

- **Put(key []byte, value []byte) error**，添加一个Put操作。
- **Delete(key []byte) error**，添加一个Delete操作。
- **Write() error**，原子性执行所有写操作。
- **Reset()**，清除所有写操作。



## 遍历: IDatabaseIterator

IDatabaseService可以创建IDatabaseIterator，后者用于遍历存储中的一批key。IDatabaseIterator使用完毕后，要由IDatabaseService销毁以释放资源。

IDatabaseIterator是单向、静态的。单向是指它只能从头向尾移动，不支持后退和随机位置的seek。静态是指遍历结果只依赖于IDatabaseIterator被创建时刻的存储状态，之后发生的存储变化不会影响遍历结果。

IDatabaseIterator不是线程安全的。多个并发routine操作同一个IDatabaseIterator是错误的，结果未知。每个routine需要创建只由自己操作的IDatabaseIterator，被多个routine创建出来、同时存在、同时遍历的若干IDatabaseIterator，不会互相影响。

### NewIterator(start []byte, limit []byte) IDatabaseIterator

NewIterator() 创建一个IDatabaseIterator，遍历那些取值范围[start, limit)的key。取值范围是前闭后开的，即要满足 start <= key < limit。如果start为nil，它表示最小key（比所有存在的key都小的key）；如果limit为nil，它表示最大key（比所有存在的key都大的key）。

创建出来的IDatabaseIterator，在调用Next()进行遍历时，会按升序返回符合条件的key。

### NewReversedIterator(start []byte, limit []byte) IDatabaseIterator

NewReversedIterator()和NewIterator()类似，不同在于：创建出来的IDatabaseIterator，在调用Next()进行遍历时，会按降序返回符合条件的key。

### DeleteIterator(it IDatabaseIterator)

销毁不再使用的IDatabaseIterator。

### IDatabaseIterator接口

IDatabaseIterator接口用于完成key的遍历。

- **Valid() bool**，当前位置是否有效，即是否可以调用Key(), Value()方法。
- **Key() ([]byte, error)**，返回当前位置的key。
- **Value() ([]byte, error)**，返回当前位置的value。
- **Next() bool**，移动到下一个位置。如果移动到的位置是个有效位置，返回true；否则返回false。

IDatabaseIterator刚刚创建时，它的位置位于第一个有效位置之前，需要Next()后才会移动到第一个位置。一般遍历循环的形式：

```go
for iter.Next() {
    // do something with iter.Key(), iter.Value()
}
```



## 嵌套交易

嵌套交易提供一种比IDatabaseBatch更灵活的批量写入方案。

IDatabaseService内部存在一个交易栈，所有操作都发生在栈顶的交易上。栈顶交易结束时，可以选择提交或者放弃数据改动，如果提交，那么所有改动会原子性地写入位于它下一层的交易。依次类推，当交易栈最后一个交易提交时，数据改动会原子性写入永久性存储。

嵌套交易主要是为了满足coschain的嵌套处理逻辑：

```go
for trx := range block.transactions {
    // 处理transaction, 任意transaction失败，当前block失败
    for op := trx.operations {
        // 处理operation，任意operation失败，当前transaction失败
    }
}
```

### BeginTransaction() 

BeginTransaction() 新创建一个交易，新交易成为栈顶交易。

### EndTransaction(commit bool) error

EndTransaction() 终止栈顶交易，commit为true时，提交改动；否则，放弃改动。

### TransactionHeight() uint

TransactionHeight()返回当前交易栈中的交易个数，即栈顶交易的高度（嵌套层数）。



## 数据回滚

数据回滚是对已经进入永久存储的数据进行回滚控制。数据回滚主要应用于coschain处理链分叉的场景，需要把已经写入永久存储的最近若干个block的改动回滚掉，然后重新应用另一分叉上的block。

IDatabaseService内部用revision标记每次原子性写入（Put/Delete/Batch），并记录相应的undo log来支持回滚。

数据回滚相关操作是线程安全的。

### GetRevision() uint64

GetRevision() 返回IDatabaseService当前的revision。

### RevertToRevision(r uint64) error

RevertToRevision() 回滚到指定的revision。 回滚一旦成功不可撤销。

### RebaseToRevision(r uint64) error

RebaseToRevision() 丢弃指定revision之前的所有undo log。这使得指定revision成为能够回滚到的最小revision。rebase一旦成功不可撤销。

如果从来不调用RebaseToRevision()，IDatabaseService可以回滚到最开始的空库状态，但实际应用中没有必要。对于已经确定的回滚下界，比如coschain的不可逆block，应该及时调用RebaseToRevision()来丢弃不需要的undo log，节省存储空间。

### Revision Tagging

revision是个IDatabaseService内部产生的uint64，没有可读性。为了方便使用，IDatabaseService支持给revision赋予string别名，即tag。

下面的tag操作是线程安全的。

- **TagRevision(r uint64, tag string) error**，给指定revision打tag。
- **GetTagRevision(tag string) (uint64, error)**，返回指定tag对应的revision。
- **GetRevisionTag(r uint64) (string, error)**，返回指定的revision的tag。
- **RevertToTag(tag string) error**，回滚到指定tag。
- **RebaseToTag(tag string) error**，rebase到指定tag。



## 重置

### DeleteAll() error

DeleteAll() 将存储重置为空，即删除所有数据，主要用于数据损坏需要根据block原始binary重新生成所有数据的场景。

DeleteAll()不是IDatabaseService的常规操作，不是线程安全的，只适合单线程调用。DeleteAll()之前，如果创建了IDatabaseIterator，必须都通过DeleteIterator()销毁掉。从DeleteAll()开始执行后，到成功返回（返回nil）前，IDatabaseService处于不可用状态，不要调用任何接口方法。

