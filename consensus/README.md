# Consensus: SABFT
self-adaptive bft
------
It generates blocks in the same manner of DPoS and adopts bft to achieve fast block confirmation. It's self adaptive in a way that it can adjust the frequency of bft process based on the load of the blockchain and network traffic.

![SABFT-flow-chart](https://github.com/coschain/contentos-go/blob/master/consensus/resource/SABFT.jpg)
## terminology
* node: a server running contentos daemon(cosd)
* block producer: a node that generate blocks
* validator: a node that participates in bft consensus
* proposer: a validator who broadcasts a proposal
* proposal: a block on which all validators try to reach consensus, once reached, it'll be committed
* commit: commit a block means mark the block as the last irreversible block

## Block generation
SABFT generates blocks in the same manner of DPoS. Each validator takes turn to produce 10 blocks in a row and a block is generated every second. Fork is possible, the longest chain is considered as the current main branch.
### Fork switch
If another branch out grows the main branch, `swichFork` is taken place.
> It finds the common ancestor of the two branches, pop all blocks on the main branch after the ancestor and apply blocks on the longer branch

------

## BFT
### 1. why we need BFT
Instant transactions are required in many scenarios, especially when it involves asset transfer.
In bitcoin world, there is no guarantee to finalize a certain block because theoretically any node with enough resource can generate a longer chain and cause a fork switch. This is a direct violation of safety in the realm of distributed system.
Hence we adopt BFT to achieve fast consensus. Once a consensus is reached on a certain block, it can never be reverted.

### 2. Performance
SABFT reaches consensus in 1~2 seconds in LAN. The bft process adopts 3-phase-commit(propose, prevote, precommit), in the propose phase, validators wait synchronously for the proposer to broadcast proposal, the rest two phases are completely asynchronous.

To better illustrate SABFT's performance, a experiment is conducted with following limitations:
>hardware information:
CPU: machdep.cpu.core_count: 2
     machdep.cpu.thread_count: 4
     machdep.cpu.brand_string: Intel(R) Core(TM) i5-5257U CPU @ 2.70GHz
RAM: 8 GB 1867 MHz DDR3

>experiment data:
consensus nodes number: 19
bandwidth limit between two nodes: 100kB/s
block size: 50kB
network latency: 0~100ms

The figure below shows the margin step of 7000 blocks. The average is 1.49.
![SABFT-margin-step](https://github.com/coschain/contentos-go/blob/master/consensus/resource/marginstep.jpg)

The figure below shows the interval between two committed blocks. The average is 1500ms.
![SABFT-commit-interval](https://github.com/coschain/contentos-go/blob/master/consensus/resource/commit_interval.jpg)

The figure below shows the interval between a block's generation time and its commit time. In average, it takes 1800ms for a block to be committed and this includes the block's generation time.
![SABFT-commit-time](https://github.com/coschain/contentos-go/blob/master/consensus/resource/commit_time.jpg)

### 3. how it's different
* SABFT's block producing process and bft process are completely decoupled. i.e. validator can generate blocks despite the state of the bft process. Let's say the current block height is 100 and the bft process only committed the block with height of 90, validator can start generating the 101's block without waiting for block 91-100 to be committed.
* Another significant difference is that the bft process does't have to reach consensus on every block. The height difference of two consecutive blocks that reached consensus is called `margin step`. It is adjusted by SABFT automatically according to the network condition and load of the Contentos chain. SABFT can usually reach consensus every 1 or 2 seconds, the margin step increases due to heavier load, network traffic or the presence of byzantine nodes.
* **tendermint/ont/neo**
Their block producing procedure are tied with bft process. During the `propose` stage of the bft, the proposer first generate a block and broadcast it. A new block can be generated only after the previous one is committed. A standard bft process generally takes a few seconds, during this time no new block can be generated which compromises the performance.
* **EOS**
EOS's approach is interesting but I personally consider it as a step back. Its implementation is called pipelined bft instead of realtime bft. The voting message of the bft process is embedded in later blocks. Say the chain has `3f+1` validators and up to `f` can be byzantine. In the 2-phase-commit step, each step requires `2f+1` votes, so the minimum time needed for the system to reach consensus is `2*(2f+1)*t`. `t` is the time span a single validator takes to generate blocks. That's why EOS needs about `2*(2*6.7+1)*6s`, `3min` to confirm a block.


### 4. behaviour
#### propose
* how to choose a proposer: a new proposer is chosen among all validators in every bft voting round in round-robin
* how to pick a proposal: proposer simply propose its head block
#### commit
it's possible that a block that is about to be committed is not on the main branch, hence a fork switch is needed.

> the voting process is completely controlled by gobft, to get more details please refer to [gobft doc](https://github.com/coschain/gobft)

### 5. self adaptation explained
In the case of network jam, validators crash or byzantine validators, block confirmation can be delayed. The self adaptive mechanism makes sure that the system can quickly confirm the latest generated block in later rounds. 

A new proposer is chosen among all validators in every bft voting round in round-robin. The proposer simply proposes the latest block it knows, when it’s confirmed, all the blocks before it will be confirmed too. In the case of network latency, other validators may not receive the proposed block or its votes. If validators always propose the latest block, bft consensus may not be reached in a very long time. To overcome this, block with smaller block number than the head block is proposed if consensus is not reached in several voting rounds. 

The bft process can be considered a state machine. The state consists of height, round and step. Step is omitted here to simplify the process. In each height, one or more rounds exist. Round starts from 0 in each height and increases if bft consensus is not reached in this round. H1R0 indicates the current state is at height of 1 and round of 0.

The following picture illustrates how SABFT adjusts its bft process if any of the abnormal situations we mentioned earlier occurs.

![SA-chart](https://github.com/coschain/contentos-go/blob/master/consensus/resource/sabft_in_general.jpg)
At t1 block 1 is generated, meanwhile the bft process starts and the proposer proposed block 1.  Soon at t1’(t1<t1’<t2), consensus is reached and block 1 is committed.  At t2, block 2 is generated and it’s proposed. However things get messy and no consensus is reached in round 0 before timeout. At t4 the state enters H2R1 and block 4 is proposed. Finally at t2’ consensus is reached on block 4 and block 2-4 is committed at once. From t6 things go back to normal and all blocks after block 5 are committed within 1 second. As is shown above the margin step in height 2 is 4, after that it quickly drops to 1.


### 6. worst case scenario
According to **FLP impossibility**, in a asynchronous network, there's no **deterministic** way to achieve consensus with one faulty process. In each round of the bft process, there always exists a critical failure like crash, network jam or malicious nodes broadcast bad message which mess up the vote process. Theoretically there's a situation where every single bft round ends up with failure but the possibility decrease exponentially as round grows. It's a bit too paranoid to worry about this.

Bottom line, there is no perfect way to guarantee both safety and liveness. We take safety as our priority, the only thing need to be worried about is that too many uncommitted blocks might eventually eat up the memory resource. But we can easily come up with a **retention policy** to discard far out blocks, which is out of the scope of this discussion.

### 7. bad behaviour punishment
Technically abnormal behaviour can be hold accountable as long as we track enough information. Here's some first thought:

* validators that are offline or constantly absent from block producing or bft voting should be removed from validator set
* validators that has following behaviour should be punished:
> 1. generates conflicting blocks
> 2. signs conflicting votes
> 3. violates the [POL](https://github.com/coschain/gobft) voting rule
> 4. constantly proposes invalid blocks

## Safety and liveness
For more information about safety and liveness, please refer to [gobft doc](https://github.com/coschain/gobft)

