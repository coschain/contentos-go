(module
 (type $FUNCSIG$viiii (func (param i32 i32 i32 i32)))
 (type $FUNCSIG$vii (func (param i32 i32)))
 (import "env" "print_string" (func $print_string (param i32 i32)))
 (import "env" "read_from_storage" (func $read_from_storage (param i32 i32 i32 i32)))
 (import "env" "save_to_storage" (func $save_to_storage (param i32 i32 i32 i32)))
 (table 0 anyfunc)
 (memory $0 1)
 (data (i32.const 4) "0@\00\00")
 (data (i32.const 16) "hello\00")
 (data (i32.const 32) "world\00")
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
     (i32.const 32)
    )
   )
  )
  (i32.store16
   (i32.add
    (i32.add
     (get_local $0)
     (i32.const 24)
    )
    (i32.const 4)
   )
   (i32.load16_u offset=20 align=1
    (i32.const 0)
   )
  )
  (i32.store offset=24
   (get_local $0)
   (i32.load offset=16 align=1
    (i32.const 0)
   )
  )
  (i32.store16
   (i32.add
    (i32.add
     (get_local $0)
     (i32.const 16)
    )
    (i32.const 4)
   )
   (i32.load16_u offset=36 align=1
    (i32.const 0)
   )
  )
  (i32.store offset=16
   (get_local $0)
   (i32.load offset=32 align=1
    (i32.const 0)
   )
  )
  (call $save_to_storage
   (i32.add
    (get_local $0)
    (i32.const 24)
   )
   (i32.const 5)
   (i32.add
    (get_local $0)
    (i32.const 16)
   )
   (i32.const 5)
  )
  (call $read_from_storage
   (i32.add
    (get_local $0)
    (i32.const 24)
   )
   (i32.const 5)
   (i32.add
    (get_local $0)
    (i32.const 11)
   )
   (i32.const 5)
  )
  (call $print_string
   (i32.add
    (get_local $0)
    (i32.const 11)
   )
   (i32.const 5)
  )
  (i32.store offset=4
   (i32.const 0)
   (i32.add
    (get_local $0)
    (i32.const 32)
   )
  )
  (i32.const 0)
 )
)
