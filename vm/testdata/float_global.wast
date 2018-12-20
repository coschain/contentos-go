(module
 (table 0 anyfunc)
 (memory $0 1)
 (data (i32.const 4) "\10@\00\00")
 (data (i32.const 12) "\c3\f5H@")
 (export "memory" (memory $0))
 (export "sub" (func $sub))
 (export "main" (func $main))
 (func $sub (param $0 i32) (result i32)
  (i32.sub
   (i32.trunc_s/f32
    (f32.load offset=12
     (i32.const 0)
    )
   )
   (get_local $0)
  )
 )
 (func $main (result i32)
  (i32.const 0)
 )
)
