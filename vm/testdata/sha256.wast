(module
 (type $FUNCSIG$viiii (func (param i32 i32 i32 i32)))
 (import "env" "sha256" (func $sha256 (param i32 i32 i32 i32)))
 (table 0 anyfunc)
 (memory $0 1)
 (data (i32.const 4) " @\00\00")
 (data (i32.const 16) "hello world\00")
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
  (i32.store
   (i32.add
    (get_local $0)
    (i32.const 44)
   )
   (i32.load offset=24 align=1
    (i32.const 0)
   )
  )
  (i64.store offset=36 align=4
   (get_local $0)
   (i64.load offset=16 align=1
    (i32.const 0)
   )
  )
  (i64.store
   (i32.add
    (get_local $0)
    (i32.const 24)
   )
   (i64.const 0)
  )
  (i64.store
   (i32.add
    (get_local $0)
    (i32.const 16)
   )
   (i64.const 0)
  )
  (i64.store offset=8
   (get_local $0)
   (i64.const 0)
  )
  (i64.store
   (get_local $0)
   (i64.const 0)
  )
  (call $sha256
   (i32.add
    (get_local $0)
    (i32.const 36)
   )
   (i32.const 11)
   (get_local $0)
   (i32.const 32)
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
