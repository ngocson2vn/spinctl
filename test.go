package main

import (
	"fmt"
	"time"
	"os"
	"strings"
	"github.com/spinctl/util"
)

func test() {
	uuid, err := util.GenerateUpperCaseUuid("test")
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	
	fmt.Println(uuid)

	tm := time.Now()
	year, month, day := tm.Date()
	hour, minute, second := tm.Clock()
	ts := tm.UnixNano() / 1e6
	fmt.Println(fmt.Sprintf("%d", ts))
	fmt.Println(fmt.Sprintf("%d%02d%02d-%02d%02d%02d", year, month, day, hour, minute, second))

	list := strings.Split("/pipelines/01DFARZDF0K3JEPRBX2WE855TT", "/")
	fmt.Println(list[2])
}
