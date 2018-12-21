(module
 (type $FUNCSIG$jii (func (param i32 i32) (result i64)))
 (type $FUNCSIG$vj (func (param i64)))
 (type $FUNCSIG$jiiii (func (param i32 i32 i32 i32) (result i64)))
 (type $FUNCSIG$viij (func (param i32 i32 i64)))
 (import "env" "contract_transfer" (func $contract_transfer (param i32 i32 i64)))
 (import "env" "get_balance_by_name" (func $get_balance_by_name (param i32 i32) (result i64)))
 (import "env" "get_contract_balance" (func $get_contract_balance (param i32 i32 i32 i32) (result i64)))
 (import "env" "print_uint64" (func $print_uint64 (param i64)))
 (table 0 anyfunc)
 (memory $0 1)
 (data (i32.const 4) "@@\00\00")
 (data (i32.const 16) "alice\00")
 (data (i32.const 32) "initminer\00")
 (data (i32.const 48) "hello\00")
 (export "memory" (memory $0))
 (export "main" (func $main))
 (func $main (result i32)
  (local $0 i32)
  (local $1 i32)
  (local $2 i32)
  (i32.store offset=4
   (i32.const 0)
   (tee_local $2
    (i32.sub
     (i32.load offset=4
      (i32.const 0)
     )
     (i32.const 48)
    )
   )
  )
  (i32.store16
   (i32.add
    (i32.add
     (get_local $2)
     (i32.const 40)
    )
    (i32.const 4)
   )
   (tee_local $0
    (i32.load16_u offset=20 align=1
     (i32.const 0)
    )
   )
  )
  (i32.store offset=40
   (get_local $2)
   (tee_local $1
    (i32.load offset=16 align=1
     (i32.const 0)
    )
   )
  )
  (call $print_uint64
   (call $get_balance_by_name
    (i32.add
     (get_local $2)
     (i32.const 40)
    )
    (i32.const 5)
   )
  )
  (i32.store16
   (i32.add
    (get_local $2)
    (i32.const 36)
   )
   (i32.load16_u offset=40 align=1
    (i32.const 0)
   )
  )
  (i64.store offset=28 align=4
   (get_local $2)
   (i64.load offset=32 align=1
    (i32.const 0)
   )
  )
  (i32.store16
   (i32.add
    (i32.add
     (get_local $2)
     (i32.const 20)
    )
    (i32.const 4)
   )
   (i32.load16_u offset=52 align=1
    (i32.const 0)
   )
  )
  (i32.store offset=20
   (get_local $2)
   (i32.load offset=48 align=1
    (i32.const 0)
   )
  )
  (call $print_uint64
   (call $get_contract_balance
    (i32.add
     (get_local $2)
     (i32.const 20)
    )
    (i32.const 5)
    (i32.add
     (get_local $2)
     (i32.const 28)
    )
    (i32.const 9)
   )
  )
  (i32.store16
   (i32.add
    (i32.add
     (get_local $2)
     (i32.const 12)
    )
    (i32.const 4)
   )
   (get_local $0)
  )
  (i32.store offset=12
   (get_local $2)
   (get_local $1)
  )
  (call $contract_transfer
   (i32.add
    (get_local $2)
    (i32.const 12)
   )
   (i32.const 5)
   (i64.const 1000)
  )
  (call $print_uint64
   (call $get_balance_by_name
    (i32.add
     (get_local $2)
     (i32.const 12)
    )
    (i32.const 5)
   )
  )
  (call $print_uint64
   (call $get_contract_balance
    (i32.add
     (get_local $2)
     (i32.const 20)
    )
    (i32.const 5)
    (i32.add
     (get_local $2)
     (i32.const 28)
    )
    (i32.const 9)
   )
  )
  (i32.store offset=4
   (i32.const 0)
   (i32.add
    (get_local $2)
    (i32.const 48)
   )
  )
  (i32.const 0)
 )
)
