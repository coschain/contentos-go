# Consensus: SABFT
self-adaptive bft
------
It generates blocks in the same manner of DPoS and adopts bft to achieve fast block confirmation. It's self adaptive in a way that it can adjust the frequency of bft process based on the load of the blockchain and network traffic.

![cmd-markdown-logo](resource/goBFT-dataflow.jpeg)
## terminology
* node: a server running contentos daemon(cosd)
* validator: a node that generates blocks and participates in bft consensus
* proposer: a validator who broadcasts a proposal
* proposal: a block on which all validators try to reach consensus, once reached, it'll be committed
* commit: commit a block means mark the block as the last irreversable block

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
Hence we adopts BFT to achieve fast consensus. Once a consensus is reached on a certain block, it can never be reverted.

### 2. Performance
SABFT reaches consensus in 1~2 seconds in LAN. The bft process adopts 3-phase-commit(propose, prevote, precommit), in the propose phase, validators wait synchronously for the proposer to broadcast proposal, the rest two phases are completely asynchronous.


### 3. how it's different
* SABFT's block produing process and bft process are completely decoupled. i.e. validator can generate blocks despite the state of the bft process. Let's say the current block height is 100 and the bft process only committed the block with height of 90, validator can start generating the 101's block without waiting for block 91-100 to be committed.
> **Comparison** with other blockchain
$$tendermint/ont/neo$$
The block producing procedure is tied with bft process. During the `propose` stage of the bft, the proposer first generate a block and broadcast it. A new block can be generated only after the previous one is committed. A standard bft process generally takes a few seconds, during this time no new block can be generated which compromises the performance.
$$EOS$$
EOS's approach is interesting but I personally consider it as a step back. Its implementation is called pipelined bft instead of realtime bft. The voting message of the bft process is embedded in later blocks. Say the chain has `3f+1` validators and up to `f` can be byzantine. In the 2-phase-commit step, each step requires `2f+1` votes, so the minimum time needed for the system to reach consensus is `2*(2f+1)*t`. `t` is the time span a single validator takes to generate blocks. That's why EOS needs about `2*(2*6.7+1)*6s`, `3min` to confirm a block.
* Another significant diffrence is that the bft process does't have to reach consensus on every block. The height difference of two consecutive blocks that reached consensus is called `margin step`. It is adjusted by SABFT automatically according to the network condition and load of the Contentos chain. SABFT can usually reach consensus every 1 or 2 seconds, the margin step increases due to heavier load, network traffic or the presence of byzantine nodes.

### 4. behaviour
#### propose
* how to choose a proposer: a new proposer is chosen among all validators in every bft voting round in round-robin manner
* how to pick a proposal: proposer simply propose its head block
#### commit
it's possible that a block that is about to be committed is not on the main branch, hence a fork switch is needed.

> the voting process is completely controlled by gobft, to get more details please refer to [gobft doc](https://github.com/coschain/gobft)

### 5. worst case scenario
According to **FLP impossibility**, in a asynchronous network, there's no **deterministic** way to achieve consensus with one faulty process. In each round of the bft process, there always exists a critical failure like crash, network jam or malicious nodes broadcast bad message which mess up the vote process. Theoretically there's a situation where every single bft round ends up failure but the possibility decrease exponentially. It's a bit too paranoid to worry about this.
Bottom line, there is no perfect way to guarentee both safety and liveness. We take safety as our priority, the only thing need to be worried about is that too many uncommitted blocks might eventually eat up the memory resource. But we can easily come up with a **retention policy** to discard far out blocks, which is out of the scope of this discussion.

### 6. bad behavious punishment
Technically abnormal behaviour can be hold accountable as long as we track enough information. Here's some first thought:

* validators that are offline or constantly absent from block producing or bft voting should be removed from validator set
* validators that has following behaviour should be punished:
> 1. generates conflicting blocks
> 2. signs conflicting votes
> 3. violates the [POL](https://github.com/coschain/gobft) voting rule
> 4. constantly proposes invalid blocks

## Safety and liveness
For more information about safety and liveness, please refer to [gobft doc](https://github.com/coschain/gobft)

