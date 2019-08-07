#include <cosiolib/contract.hpp>
#include <cosiolib/print.hpp>

struct voter {
    voter():name(""),haveVoted(false){}
    std::string name;
    bool haveVoted;

    COSIO_SERIALIZE(voter,(name)(haveVoted))
};

struct freezeinfo {
    freezeinfo():id(0),op(0),agree(0),accounts(std::vector<std::string>()),memos(std::vector<std::string>()),producers(std::vector<voter>()){}
    uint32_t id; // proposal unique id
    uint32_t op; // freeze=1 or unfreeze=0
    uint32_t agree; // total agrees to the proposal
    std::vector<std::string> accounts; // accounts to be freeze or unfreeze
    std::vector<std::string> memos; // the freeze memo
    std::vector<voter> producers; // all producers when proposal

    COSIO_SERIALIZE(freezeinfo,(id)(op)(agree)(accounts)(memos)(producers))
};

struct idinfo : public cosio::singleton_record {
    idinfo():proposal_id(0){}

    uint32_t proposal_id; // proposal global id
    COSIO_SERIALIZE_DERIVED(idinfo,cosio::singleton_record,(proposal_id))
};

class freeze : public cosio::contract {
public:
    using cosio::contract::contract;

    void proposalfreeze(const std::vector<std::string>& accounts, int32_t op ,const std::vector<std::string>& memos) {
        cosio::cosio_assert(op==1 || op==0, std::string("op invalid freeze=1 or unfreeze=0"));

        for(int i=0; i<accounts.size();i++){
            // make sure user exist
            cosio::cosio_assert(cosio::user_exist(accounts[i]),std::string("proposal user not exist:")+accounts[i]);
        }

        auto caller = cosio::get_contract_caller();
        auto producers = cosio::block_producers();

        // only bp can send proposal
        std::vector<std::string>::const_iterator it = std::find(producers.begin(),producers.end(),caller.string());
        if(it == producers.end()){
            cosio::cosio_assert(false, std::string("caller is not producers, name:") + caller.string());
        }

        auto r = pid.get_or_create();

        if(freezetable.has(r.proposal_id)) {
            cosio::cosio_assert(false, std::string("proposal duplicated"));
        } else {
            // add proposal info
            freezetable.insert([&](freezeinfo& f){
                f.id = r.proposal_id;
                f.agree = 0;
                f.accounts = accounts;
                f.op = op;
                f.memos = memos;
                for(int i=0;i<producers.size();i++) {
                    voter v;
                    v.name = producers[i];
                    v.haveVoted = false;
                    f.producers.push_back(v);
                }
            });

            // increase proposal id
            pid.update([&](idinfo &i){
                i.proposal_id++;
            });
        }
    }

    void vote(uint32_t id) {
        auto r = pid.get_or_create();
        cosio::cosio_assert(id < r.proposal_id, std::string("proposal id exceed"));
        auto caller = cosio::get_contract_caller();

        auto name = caller.string();
        auto num = cosio::current_block_number();

        // process vote
        if(freezetable.has(id)) {
            freezetable.update(id,[&](freezeinfo& f){
                std::vector<voter>::iterator it = std::find_if(f.producers.begin(),f.producers.end(),[&name](const voter& vv){return vv.name == name;});
                if(it == f.producers.end()) {
                    cosio::cosio_assert(false, std::string("caller is not in producers when proposal, caller:") + name);
                }
                if(it->haveVoted) {
                    cosio::cosio_assert(false, std::string("caller has voted, caller:") + name);
                }
                it->haveVoted = true;
                f.agree++;
            });
        } else {
            cosio::cosio_assert(false, std::string("id not exist"));
        }

        auto v = freezetable.get(id);
        auto all_producer_size = v.producers.size();
        if(all_producer_size < 3) {
            all_producer_size = 3;
        }
        auto limit = (all_producer_size/3)*2;
        
        // setafreeze if most bp agree
        if(v.agree > limit) {
            cosio::update_freeze(v.accounts,v.op,v.memos);
            cosio::print_f("freeze proposal % has been executed, then remove it from contract storage. \n",id);
            // this proposal is done, remove it
            freezetable.remove(id);
        }
    }

private:

    COSIO_DEFINE_TABLE( freezetable, freezeinfo, (id) );
    COSIO_DEFINE_NAMED_SINGLETON( pid, "proposalid", idinfo);
};

COSIO_ABI(freeze, (proposalfreeze)(vote))
