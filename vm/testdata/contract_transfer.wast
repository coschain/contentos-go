(module
 (type $FUNCSIG$jii (func (param i32 i32) (result i64)))
 (type $FUNCSIG$vj (func (param i64)))
 (type $FUNCSIG$jiiii (func (param i32 i32 i32 i32) (result i64)))
 (type $FUNCSIG$viijii (func (param i32 i32 i64 i32 i32)))
 (import "env" "get_balance_by_name" (func $get_balance_by_name (param i32 i32) (result i64)))
 (import "env" "get_contract_balance" (func $get_contract_balance (param i32 i32 i32 i32) (result i64)))
 (import "env" "print_uint" (func $print_uint (param i64)))
 (import "env" "transfer" (func $transfer (param i32 i32 i64 i32 i32)))
 (table 0 anyfunc)
 (memory $0 1)
 (data (i32.const 4) "`@\00\00")
 (data (i32.const 16) "alice\00")
 (data (i32.const 32) "initminer\00")
 (data (i32.const 48) "hello\00")
 (data (i32.const 64) "alice\00")
 (data (i32.const 80) "hello\00")
 (export "memory" (memory $0))
 (export "main" (func $main))
 (func $main (result i32)
  (local $0 i64)
  (local $1 i32)
  (i32.store offset=4
   (i32.const 0)
   (tee_local $1
    (i32.sub
     (i32.load offset=4
      (i32.const 0)
     )
     (i32.const 64)
    )
   )
  )
  (i32.store16
   (i32.add
    (i32.add
     (get_local $1)
     (i32.const 56)
    )
    (i32.const 4)
   )
   (i32.load16_u offset=20 align=1
    (i32.const 0)
   )
  )
  (i32.store offset=56
   (get_local $1)
   (i32.load offset=16 align=1
    (i32.const 0)
   )
  )
  (i64.store offset=48
   (get_local $1)
   (tee_local $0
    (call $get_balance_by_name
     (i32.add
      (get_local $1)
      (i32.const 56)
     )
     (i32.const 5)
    )
   )
  )
  (call $print_uint
   (get_local $0)
  )
  (i32.store16
   (i32.add
    (get_local $1)
    (i32.const 44)
   )
   (i32.load16_u offset=40 align=1
    (i32.const 0)
   )
  )
  (i64.store offset=36 align=4
   (get_local $1)
   (i64.load offset=32 align=1
    (i32.const 0)
   )
  )
  (i32.store16
   (i32.add
    (i32.add
     (get_local $1)
     (i32.const 28)
    )
    (i32.const 4)
   )
   (i32.load16_u offset=52 align=1
    (i32.const 0)
   )
  )
  (i32.store offset=28
   (get_local $1)
   (i32.load offset=48 align=1
    (i32.const 0)
   )
  )
  (i64.store offset=16
   (get_local $1)
   (tee_local $0
    (call $get_contract_balance
     (i32.add
      (get_local $1)
      (i32.const 28)
     )
     (i32.const 5)
     (i32.add
      (get_local $1)
      (i32.const 36)
     )
     (i32.const 9)
    )
   )
  )
  (call $print_uint
   (get_local $0)
  )
  (i32.store16
   (i32.add
    (i32.add
     (get_local $1)
     (i32.const 8)
    )
    (i32.const 4)
   )
   (i32.load16_u offset=68 align=1
    (i32.const 0)
   )
  )
  (i32.store16
   (i32.add
    (get_local $1)
    (i32.const 4)
   )
   (i32.load16_u offset=84 align=1
    (i32.const 0)
   )
  )
  (i32.store offset=8
   (get_local $1)
   (i32.load offset=64 align=1
    (i32.const 0)
   )
  )
  (i32.store
   (get_local $1)
   (i32.load offset=80 align=1
    (i32.const 0)
   )
  )
  (call $transfer
   (i32.add
    (get_local $1)
    (i32.const 8)
   )
   (i32.const 5)
   (i64.const 1000)
   (get_local $1)
   (i32.const 5)
  )
  (i64.store offset=48
   (get_local $1)
   (tee_local $0
    (call $get_balance_by_name
     (i32.add
      (get_local $1)
      (i32.const 8)
     )
     (i32.const 5)
    )
   )
  )
  (call $print_uint
   (get_local $0)
  )
  (i64.store offset=16
   (get_local $1)
   (tee_local $0
    (call $get_contract_balance
     (i32.add
      (get_local $1)
      (i32.const 28)
     )
     (i32.const 5)
     (i32.add
      (get_local $1)
      (i32.const 36)
     )
     (i32.const 9)
    )
   )
  )
  (call $print_uint
   (get_local $0)
  )
  (i32.store offset=4
   (i32.const 0)
   (i32.add
    (get_local $1)
    (i32.const 64)
   )
  )
  (i32.const 0)
 )
)
