package request

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

var nameLib = "abcdefghijklmnopqrstuvwxyz01234567890"

func makeRequest(rawStr string) string {
	idx := strings.Index(rawStr, " ")
	cmdType := rawStr[:idx]
	switch cmdType {
	case "create":
		return makeCreateAccount(rawStr)
	case "transfer":
		return makeTransfer(rawStr)
	case "post":
		return makePostArticle(rawStr)
	case "follow":
		return rawStr
	}
	return""
}

func makeCreateAccount(rawStr string) string {
	suffix := ""
	for i:=0;i<10;i++{
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		idx := r.Intn(len(nameLib))
		suffix += string(nameLib[idx])
	}
	return fmt.Sprintf(rawStr, suffix)
}

func makeTransfer(rawStr string) string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	value := r.Intn(10)
	return fmt.Sprintf(rawStr, 1+value)
}

func makePostArticle(rawStr string) string {
	var tag = ""
	var title = ""
	var content = ""
	for i:=0;i<10;i++ {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		idx := r.Intn(len(nameLib))
		tag += string(nameLib[idx])
	}
	for i:=0;i<20;i++ {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		idx := r.Intn(len(nameLib))
		title += string(nameLib[idx])
	}
	for i:=0;i<1024;i++ {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		idx := r.Intn(len(nameLib))
		content += string(nameLib[idx])
	}
	return fmt.Sprintf(rawStr, tag, title, content)
}