(module
 (type $FUNCSIG$viii (func (param i32 i32 i32)))
 (type $FUNCSIG$vii (func (param i32 i32)))
 (import "env" "cos_assert" (func $cos_assert (param i32 i32 i32)))
 (import "env" "print_string" (func $print_string (param i32 i32)))
 (table 0 anyfunc)
 (memory $0 1)
 (data (i32.const 4) "@@\00\00")
 (data (i32.const 16) "assert error\00")
 (data (i32.const 32) "should not be printed\00")
 (export "memory" (memory $0))
 (export "main" (func $main))
 (func $main (result i32)
  (local $0 i32)
  (i32.store offset=4
   (i32.const 0)
   (tee_local $0
    (i32.sub
     (i32.load offset=4
      (i32.const 0)
     )
     (i32.const 48)
    )
   )
  )
  (i32.store8
   (i32.add
    (get_local $0)
    (i32.const 44)
   )
   (i32.load8_u offset=28
    (i32.const 0)
   )
  )
  (i32.store
   (i32.add
    (get_local $0)
    (i32.const 40)
   )
   (i32.load offset=24 align=1
    (i32.const 0)
   )
  )
  (i64.store offset=32 align=4
   (get_local $0)
   (i64.load offset=16 align=1
    (i32.const 0)
   )
  )
  (i32.store16
   (i32.add
    (get_local $0)
    (i32.const 20)
   )
   (i32.load16_u offset=52
    (i32.const 0)
   )
  )
  (i32.store
   (i32.add
    (get_local $0)
    (i32.const 16)
   )
   (i32.load offset=48
    (i32.const 0)
   )
  )
  (i64.store offset=8
   (get_local $0)
   (i64.load offset=40
    (i32.const 0)
   )
  )
  (i64.store
   (get_local $0)
   (i64.load offset=32
    (i32.const 0)
   )
  )
  (call $cos_assert
   (i32.const 0)
   (i32.add
    (get_local $0)
    (i32.const 32)
   )
   (i32.const 20)
  )
  (call $print_string
   (get_local $0)
   (i32.const 30)
  )
  (i32.store offset=4
   (i32.const 0)
   (i32.add
    (get_local $0)
    (i32.const 48)
   )
  )
  (i32.const 0)
 )
)
