#include <cosiolib/contract.hpp>
#include <cosiolib/print.hpp>

const uint32_t expire_blocks = 300;

struct voter {
    voter():name(""),haveVoted(false){}
    std::string name;
    bool haveVoted;

    COSIO_SERIALIZE(voter,(name)(haveVoted))
};

struct item : public cosio::singleton_record {
    item(): admin(""), agree(0), begin_block(0), producers(std::vector<voter>()) {}

    std::string admin; // the proposal target
    uint32_t agree; // total agrees to the proposal
    uint64_t begin_block; // when expired, this item become invalid, we accept new proposal
    std::vector<voter> producers; // all producers when proposal

    COSIO_SERIALIZE_DERIVED(item,cosio::singleton_record,(admin)(agree)(begin_block)(producers))
};

class copyright : public cosio::contract {
public:
    using cosio::contract::contract;

	void setadmin( const std::string& user ) {
		cosio::set_copyright_admin(user);
	}

	void setcopyright( const uint64_t postid, int32_t copyright, const std::string& memo ) {
		cosio::update_copyright(postid, copyright, memo);
	}

	void setcopyrights( const std::vector<uint64_t>& postids, const std::vector<int32_t>& copyrights, const std::vector<std::string>& memos ) {
		cosio::update_copyrights(postids, copyrights, memos);
	}

    void proposal(const std::string& user) {
        // make sure user exist
        cosio::cosio_assert(cosio::user_exist(user), std::string("proposal user not exist:")+user);

        auto caller = cosio::get_contract_caller();
        auto producers = cosio::block_producers();

        std::vector<std::string>::const_iterator it = std::find(producers.begin(),producers.end(),caller.string());
        if(it == producers.end()){
            cosio::cosio_assert(false, std::string("caller is not producers, name:") + caller.string());
        }

        // check expire
        auto num = cosio::current_block_number();
        if (box.exists()) {
            auto v = box.get();
            if(v.begin_block + expire_blocks > num){
                cosio::cosio_assert(false, std::string("last proposal still available"));
            }
        }

        // a new proposal
        box.get_or_create();
        box.update([&](item &vt){
                vt.admin = user;
                vt.agree = 0;
                vt.begin_block = num;
                vt.producers.clear();
                for(int i=0;i<producers.size();i++) {
                    voter v;
                    v.name = producers[i];
                    v.haveVoted = false;
                    vt.producers.push_back(v);
                }
        });
    }

    void vote() {
        auto caller = cosio::get_contract_caller();
        if(!box.exists()) {
            cosio::cosio_assert(false, std::string("no proposal yet"));
        }

        auto name = caller.string();
        auto num = cosio::current_block_number();

        // add vote
        box.update([&](item &vt){
            // check expire
            if(vt.begin_block + expire_blocks <= num) {
                cosio::cosio_assert(false, std::string("proposal expired, please commit a new proposal"));
            }
            // check if caller in the producer list when proposal created
            std::vector<voter>::iterator it = std::find_if(vt.producers.begin(),vt.producers.end(),[&name](const voter& vv){return vv.name == name;});
            if(it == vt.producers.end()) {
                cosio::cosio_assert(false, std::string("caller is not in producers when proposal, caller:") + name);
            }
            // vote only once
            if(it->haveVoted) {
                cosio::cosio_assert(false, std::string("caller has voted, caller:") + name);
            }
            it->haveVoted = true;
            vt.agree++;
        });

        auto v = box.get();
        auto all_producer_size = v.producers.size();
        if(all_producer_size < 3) {
            all_producer_size = 3;
        }
        auto limit = (all_producer_size/3)*2;

        // setadmin if most bp agree
        if(v.agree > limit) {
            setadmin(v.admin);
            cosio::print("copyright admin proposal has been executed, reset proposal. \n");
            box.remove();
        }
    }

private:

    COSIO_DEFINE_NAMED_SINGLETON( box, "electionbox", item);
};


COSIO_ABI(copyright, (setcopyright)(setcopyrights)(proposal)(vote))