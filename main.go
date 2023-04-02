package main

import (
	"context"
	"flag"
	"math/rand"
	"snakegame/internal"
	"time"

	"github.com/golang/glog"
)

func main() {
	flag.Parse()
	defer glog.Flush()
	rand.Seed(time.Now().Unix())
	ctx := context.Background()
	glog.Infof("Starting loop")
	if err := internal.Loop(ctx); err != nil {
		glog.Fatal(err)
	}
}
