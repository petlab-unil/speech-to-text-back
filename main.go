package main

import (
	"context"
	"log"
	"os"
	"s2t/stream"
)

func main() {

	audioFile := "enzo.flac"

	ctx := context.Background()
	file, err := os.Open(audioFile)

	if err != nil {
		log.Fatal(err)
	}

	defer file.Close()
	s := stream.NewStream(ctx, file)
	s.Start()
}
