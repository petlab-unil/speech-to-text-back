package stream

import (
	speech "cloud.google.com/go/speech/apiv1"
	"context"
	"fmt"
	"github.com/cheggaaa/pb/v3"
	speechpb "google.golang.org/genproto/googleapis/cloud/speech/v1"
	"io"
	"log"
	"os"
	"sync"
	"time"
)

type Stream struct {
	ctx          context.Context
	speechClient *speech.Client
	speechStream speechpb.Speech_StreamingRecognizeClient
	file         *os.File
	fileBuffer   []byte
	progressBar  *pb.ProgressBar
	mutex        *sync.Mutex
}

func NewStream(ctx context.Context, file *os.File) Stream {
	client, err := speech.NewClient(ctx)
	if err != nil {
		log.Fatal(err)
	}

	speechStream, err := client.StreamingRecognize(ctx)
	if err != nil {
		log.Fatal(err)
	}

	fileStat, _ := file.Stat()
	fileSize := fileStat.Size()

	progressBar := pb.StartNew(int(fileSize / 32000))

	fileBuffer := make([]byte, 32000)

	stream := Stream{
		ctx:          ctx,
		speechClient: client,
		speechStream: speechStream,
		file:         file,
		fileBuffer:   fileBuffer,
		progressBar:  progressBar,
		mutex:        &sync.Mutex{},
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
	for {
		n, err := s.file.Read(s.fileBuffer)
		s.mutex.Lock()
		if n > 0 {
			if err := s.speechStream.Send(&speechpb.StreamingRecognizeRequest{
				StreamingRequest: &speechpb.StreamingRecognizeRequest_AudioContent{
					AudioContent: s.fileBuffer[:n],
				},
			}); err != nil {
				log.Printf("Could not send audio: %v", err)
			}
		}
		if err == io.EOF {
			// Nothing else to pipe, close the stream.
			if err := s.speechStream.CloseSend(); err != nil {
				log.Fatalf("Could not close stream: %v", err)
			}
			s.mutex.Unlock()
			return
		}
		if err != nil {
			log.Printf("Could not read from file: %v", err)
		}
		s.progressBar.Increment()
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
			fmt.Printf("Result: %+v\n", result)
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
