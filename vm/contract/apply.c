
#include "api.h"

extern "C" void apply( uint32 method ){
    char caller[16];
    char owner[16];
    char contract[16];
    int length = read_contract_op_params_length();

    read_contract_caller(caller, 16);
    read_contract_owner(caller, 16);

    char params[length];
    read_contract_op_params(params, length);
}