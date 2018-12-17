(module
 (type $FUNCSIG$iii (func (param i32 i32) (result i32)))
 (import "env" "writet1" (func $writet1 (param i32 i32) (result i32)))
 (table 0 anyfunc)
 (memory $0 1)
 (data (i32.const 4) " @\00\00")
 (data (i32.const 16) "hello\00 world\00")
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
     (i32.const 16)
    )
   )
  )
  (i32.store8
   (i32.add
    (get_local $1)
    (i32.const 12)
   )
   (i32.load8_u offset=28
    (i32.const 0)
   )
  )
  (i32.store
   (i32.add
    (get_local $1)
    (i32.const 8)
   )
   (i32.load offset=24 align=1
    (i32.const 0)
   )
  )
  (i64.store align=4
   (get_local $1)
   (i64.load offset=16 align=1
    (i32.const 0)
   )
  )
  (set_local $0
   (call $writet1
    (get_local $1)
    (get_local $1)
   )
  )
  (i32.store offset=4
   (i32.const 0)
   (i32.add
    (get_local $1)
    (i32.const 16)
   )
  )
  (get_local $0)
 )
)
