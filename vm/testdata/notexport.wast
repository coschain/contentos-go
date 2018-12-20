(module
 (type $FUNCSIG$iii (func (param i32 i32) (result i32)))
 (import "env" "add" (func $add (param i32 i32) (result i32)))
 (table 0 anyfunc)
 (memory $0 1)
 (data (i32.const 4) "\10@\00\00")
 (export "memory" (memory $0))
 (export "main" (func $main))
 (func $main (result i32)
  (drop
   (call $add
    (i32.const 1)
    (i32.const 2)
   )
  )
  (i32.const 0)
 )
)
