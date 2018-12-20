(module
 (type $FUNCSIG$fff (func (param f32 f32) (result f32)))
 (import "env" "add" (func $add (param f32 f32) (result f32)))
 (table 0 anyfunc)
 (memory $0 1)
 (data (i32.const 4) "\10@\00\00")
 (export "memory" (memory $0))
 (export "main" (func $main))
 (func $main (result i32)
  (drop
   (call $add
    (f32.const 3.140000104904175)
    (f32.const 1.2000000476837158)
   )
  )
  (i32.const 0)
 )
)
