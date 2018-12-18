(module
 (type $FUNCSIG$j (func (result i64)))
 (type $FUNCSIG$vj (func (param i64)))
 (type $FUNCSIG$ii (func (param i32) (result i32)))
 (type $FUNCSIG$vi (func (param i32)))
 (type $FUNCSIG$vii (func (param i32 i32)))
 (import "env" "current_block_number" (func $current_block_number (result i64)))
 (import "env" "current_timestamp" (func $current_timestamp (result i64)))
 (import "env" "current_witness" (func $current_witness (param i32) (result i32)))
 (import "env" "print_string" (func $print_string (param i32 i32)))
 (import "env" "print_uint32" (func $print_uint32 (param i32)))
 (import "env" "print_uint64" (func $print_uint64 (param i64)))
 (table 0 anyfunc)
 (memory $0 1)
 (data (i32.const 4) "\10@\00\00")
 (export "memory" (memory $0))
 (export "main" (func $main))
 (func $main (result i32)
  (local $0 i32)
  (local $1 i32)
  (i32.store offset=4
   (i32.const 0)
   (tee_local $1
    (i32.sub
     (i32.load offset=4
      (i32.const 0)
     )
     (i32.const 32)
    )
   )
  )
  (call $print_uint64
   (call $current_block_number)
  )
  (call $print_uint64
   (call $current_timestamp)
  )
  (call $print_uint32
   (tee_local $0
    (call $current_witness
     (get_local $1)
    )
   )
  )
  (call $print_string
   (get_local $1)
   (get_local $0)
  )
  (i32.store offset=4
   (i32.const 0)
   (i32.add
    (get_local $1)
    (i32.const 32)
   )
  )
  (i32.const 0)
 )
)
