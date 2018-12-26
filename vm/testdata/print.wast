(module
 (type $FUNCSIG$vii (func (param i32 i32)))
 (type $FUNCSIG$vi (func (param i32)))
 (import "env" "print_str" (func $print_str (param i32 i32)))
 (import "env" "print_uint" (func $print_uint (param i32)))
 (table 0 anyfunc)
 (memory $0 1)
 (data (i32.const 4) " @\00\00")
 (data (i32.const 16) "hello world\n\00")
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
     (i32.const 16)
    )
   )
  )
  (i32.store8
   (i32.add
    (get_local $0)
    (i32.const 12)
   )
   (i32.load8_u offset=28
    (i32.const 0)
   )
  )
  (i32.store
   (i32.add
    (get_local $0)
    (i32.const 8)
   )
   (i32.load offset=24 align=1
    (i32.const 0)
   )
  )
  (i64.store align=4
   (get_local $0)
   (i64.load offset=16 align=1
    (i32.const 0)
   )
  )
  (call $print_str
   (get_local $0)
   (i32.const 11)
  )
  (call $print_uint
   (i32.const 42)
  )
  (i32.store offset=4
   (i32.const 0)
   (i32.add
    (get_local $0)
    (i32.const 16)
   )
  )
  (i32.const 0)
 )
)
