package main

import "fmt"

func the_interface(s *MyStruct) interface{} {
	return s
}

type MyInterface interface {
	GetName() string
}

type MyStruct struct {
	name string
}

func (s *MyStruct) GetName() string {
	return s.name
}

func main() {
	//var b *MyInterface

	a := &MyStruct{name: "hehe"}
	c := the_interface(a)
	d := c.(MyInterface)
	fmt.Println(d.GetName())
	a.name = "haha"
	fmt.Println(d.GetName())
	//i := the_interfece("haha")
	//a := reflect.ValueOf(i).Elem()
	//fmt.Printf("%T", a)
	//elem := reflect.ValueOf(b).Elem()
	//elem.Set(reflect.ValueOf(a))
	//fmt.Printf("%T", elem)

	//a := reflect.TypeOf(MyInterface())
	//fmt.Printf("%T", a.Type())
}
