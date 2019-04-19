// +build ignore

package main

import (
	"github.com/shurcooL/vfsgen"
	"log"
	"net/http"
)

func main() {
	err := vfsgen.Generate(
		http.Dir("./assets"),
		vfsgen.Options{
			Filename:     "./assets.go",
			PackageName:  "main",
			VariableName: "assets",
		})
	if err != nil {
		log.Println(err)
	}
}
