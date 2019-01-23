(module
 (type $FUNCSIG$jii (func (param i32 i32) (result i64)))
 (type $FUNCSIG$vj (func (param i64)))
 (type $FUNCSIG$jiiii (func (param i32 i32 i32 i32) (result i64)))
 (type $FUNCSIG$vii (func (param i32 i32)))
 (type $FUNCSIG$j (func (result i64)))
 (import "env" "get_contract_balance" (func $get_contract_balance (param i32 i32 i32 i32) (result i64)))
 (import "env" "get_user_balance" (func $get_user_balance (param i32 i32) (result i64)))
 (import "env" "print_str" (func $print_str (param i32 i32)))
 (import "env" "print_uint" (func $print_uint (param i64)))
 (import "env" "read_contract_caller" (func $read_contract_caller (param i32 i32)))
 (import "env" "read_contract_owner" (func $read_contract_owner (param i32 i32)))
 (import "env" "read_contract_sender_value" (func $read_contract_sender_value (result i64)))
 (table 0 anyfunc)
 (memory $0 1)
 (data (i32.const 4) "0@\00\00")
 (data (i32.const 16) "initminer\00")
 (data (i32.const 32) "hello\00")
 (export "memory" (memory $0))
 (export "main" (func $main))
 (func $main (result i32)
  (local $0 i32)
  (local $1 i64)
  (local $2 i32)
  (i32.store offset=4
   (i32.const 0)
   (tee_local $2
    (i32.sub
     (i32.load offset=4
      (i32.const 0)
     )
     (i32.const 96)
    )
   )
  )
  (i32.store16
   (i32.add
    (i32.add
     (get_local $2)
     (i32.const 84)
    )
    (i32.const 8)
   )
   (tee_local $0
    (i32.load16_u offset=24 align=1
     (i32.const 0)
    )
   )
  )
  (i64.store offset=84 align=4
   (get_local $2)
   (tee_local $1
    (i64.load offset=16 align=1
     (i32.const 0)
    )
   )
  )
  (call $print_uint
   (call $get_user_balance
    (i32.add
     (get_local $2)
     (i32.const 84)
    )
    (i32.const 9)
   )
  )
  (i32.store16
   (i32.add
    (i32.add
     (get_local $2)
     (i32.const 72)
    )
    (i32.const 8)
   )
   (get_local $0)
  )
  (i64.store offset=72 align=4
   (get_local $2)
   (get_local $1)
  )
  (i32.store16
   (i32.add
    (get_local $2)
    (i32.const 68)
   )
   (i32.load16_u offset=36 align=1
    (i32.const 0)
   )
  )
  (i32.store offset=64
   (get_local $2)
   (i32.load offset=32 align=1
    (i32.const 0)
   )
  )
  (call $print_uint
   (call $get_contract_balance
    (i32.add
     (get_local $2)
     (i32.const 64)
    )
    (i32.const 5)
    (i32.add
     (get_local $2)
     (i32.const 72)
    )
    (i32.const 9)
   )
  )
  (call $read_contract_owner
   (i32.add
    (get_local $2)
    (i32.const 32)
   )
   (i32.const 20)
  )
  (call $print_str
   (i32.add
    (get_local $2)
    (i32.const 32)
   )
   (i32.const 20)
  )
  (call $read_contract_caller
   (get_local $2)
   (i32.const 20)
  )
  (call $print_str
   (get_local $2)
   (i32.const 20)
  )
  (call $print_uint
   (call $read_contract_sender_value)
  )
  (i32.store offset=4
   (i32.const 0)
   (i32.add
    (get_local $2)
    (i32.const 96)
   )
  )
  (i32.const 0)
 )
)
