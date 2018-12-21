package abi

import "encoding/json"

type JsonABI struct {
	Version 	string				`json:"version"`
	Types 		[]jsonAbiTypedef	`json:"types"`
	Structs 	[]jsonAbiStruct		`json:"structs"`
	Methods 	[]jsonAbiMethod		`json:"actions"`
	Tables 		[]jsonAbiTable		`json:"tables"`
}

type jsonAbiTypedef struct {
	Name 	string		`json:"new_type_name"`
	Type 	string		`json:"type"`
}

type jsonAbiStruct struct {
	Name 	string					`json:"name"`
	Base 	string					`json:"base"`
	Fields 	[]jsonAbiStructField	`json:"fields"`
}

type jsonAbiStructField struct {
	Name 	string		`json:"name"`
	Type 	string		`json:"type"`
}

type jsonAbiMethod struct {
	Name 	string		`json:"name"`
	Type 	string		`json:"type"`
}

type jsonAbiTable struct {
	Name 		string		`json:"name"`
	Type 		string		`json:"type"`
	Primary 	string		`json:"primary"`
	Secondary	[]string	`json:"secondary"`
}

func (abi *JsonABI) Marshal() ([]byte, error) {
	return json.Marshal(abi)
}

func (abi *JsonABI) Unmarshal(jsonData []byte) error {
	output := new(JsonABI)
	if err := json.Unmarshal(jsonData, output); err != nil {
		return err
	}
	*abi = *output
	return nil
}
