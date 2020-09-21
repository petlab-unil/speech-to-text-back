package Speech2Text

import (
	speech "cloud.google.com/go/speech/apiv1"
	"context"
	"encoding/json"
	"fmt"
	speechpb "google.golang.org/genproto/googleapis/cloud/speech/v1"
	"io"
	"log"
	"sync"
	"time"
)

type Stream struct {
	ctx          context.Context
	speechClient *speech.Client
	speechStream speechpb.Speech_StreamingRecognizeClient
	fileBuffer   chan []byte
	StreamResp   chan []byte
	mutex        *sync.Mutex
	size         int
}

func NewStream(ctx context.Context, fileBuffer chan []byte, size int) Stream {
	client, err := speech.NewClient(ctx)
	if err != nil {
		log.Fatal(err)
	}

	speechStream, err := client.StreamingRecognize(ctx)
	if err != nil {
		log.Fatal(err)
	}

	stream := Stream{
		ctx:          ctx,
		speechClient: client,
		speechStream: speechStream,
		fileBuffer:   fileBuffer,
		StreamResp:   make(chan []byte),
		mutex:        &sync.Mutex{},
		size:         size,
	}

	return stream
}

func (s *Stream) InitStream() {
	if err := s.speechStream.Send(&speechpb.StreamingRecognizeRequest{
		StreamingRequest: &speechpb.StreamingRecognizeRequest_StreamingConfig{
			StreamingConfig: &speechpb.StreamingRecognitionConfig{
				Config: &speechpb.RecognitionConfig{
					Encoding:        speechpb.RecognitionConfig_FLAC,
					SampleRateHertz: 32000,
					LanguageCode:    "fr-FR",
				},
			},
		},
	}); err != nil {
		log.Fatal(err)
	}
}

func (s *Stream) StartStream() {
	currentSize := 0
	for {
		fileBuffer := <-s.fileBuffer
		currentSize += len(fileBuffer)
		s.mutex.Lock()
		if len(fileBuffer) > 0 {
			if err := s.speechStream.Send(&speechpb.StreamingRecognizeRequest{
				StreamingRequest: &speechpb.StreamingRecognizeRequest_AudioContent{
					AudioContent: fileBuffer,
				},
			}); err != nil {
				log.Printf("Could not send audio: %v", err)
			}
		} else {
			_ = s.speechStream.CloseSend()
			break
		}
		if currentSize >= s.size {
			_ = s.speechStream.CloseSend()
			break
		}
		s.mutex.Unlock()
	}
}

func (s *Stream) Listen(done chan bool) {
	for {
		resp, err := s.speechStream.Recv()
		if err == io.EOF {
			fmt.Printf("EOF\n")
			break
		}
		if err != nil {
			log.Fatalf("Cannot stream results: %v", err)
		}
		if err := resp.Error; err != nil {
			log.Fatalf("Could not recognize: %v", err)
		}
		for _, result := range resp.Results {
			serialized, _ := json.Marshal(result)
			s.StreamResp <- serialized
		}
	}
	done <- true
}

func (s *Stream) Start() {
	s.InitStream()
	go s.StartStream()
	done := make(chan bool)
	go s.Listen(done)
	go func() {
		for {
			duration := 20 * time.Second
			time.Sleep(duration)
			s.mutex.Lock()
			_ = s.speechStream.CloseSend()
			speechStream, err := s.speechClient.StreamingRecognize(s.ctx)
			if err != nil {
				log.Fatal(err)
			}
			s.speechStream = speechStream
			s.InitStream()
			s.mutex.Unlock()
		}
	}()
	<-done
}
