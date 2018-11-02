package main

import (
	"fmt"
	"os"
)
import "html/template"

func main()  {
	tmpl := `
type {{ . }} struct {
	dba 		storage.Database
	mainKey 	*base.AccountName
}

func NewSoAccountWrap(dba storage.Database, key *base.AccountName) *SoAccountWrap{
	result := &SoAccountWrap{ dba, key}
	return result
}
`

	t := template.New("layout.html")
	t, _ = t.Parse(tmpl)
	fmt.Println(t.Name())
	t.Execute( os.Stdout, "Hello World")

}