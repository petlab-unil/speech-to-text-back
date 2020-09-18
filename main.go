package main

import (
	"context"
	"log"
	"s2t/stream"
	"os"
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
	s.InitStream()
	go s.StartStream()
	go s.RestartDaemon()
	s.Listen()
}
