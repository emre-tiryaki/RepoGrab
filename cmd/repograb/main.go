package main

import (
	"fmt"
	"log"

	"github.com/emre-tiryaki/repograb/internal/provider"
)

func main(){
	p := &provider.GithubProvider{Token: ""}

	items, err := p.FetchTree("torvalds", "linux", "master", "")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("files")
	for _, item := range items {
		fmt.Printf("[%s] %s\n", item.Type, item.Name)
	}
}
