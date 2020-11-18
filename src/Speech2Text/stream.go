package Speech2Text

import (
	speech "cloud.google.com/go/speech/apiv1"
	"context"
	"encoding/json"
	"fmt"
	speechpb "google.golang.org/genproto/googleapis/cloud/speech/v1"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"io"
	"log"
	"speech-to-text-back/src/server/account"
	"sync"
	"time"
)

type Stream struct {
	ctx             context.Context
	speechClient    *speech.Client
	speechStream    speechpb.Speech_StreamingRecognizeClient
	fileBuffer      chan []byte
	StreamResp      chan []byte
	StreamErr       chan []byte
	Closed          bool
	mutex           *sync.Mutex
	size            int
	inputEOF        bool
	mongoSession    *mgo.Session
	translation     *account.Translation
	sampleRateHertz int32
	language        string
	audioType       speechpb.RecognitionConfig_AudioEncoding
	model           string
}

func NewStream(ctx context.Context,
	fileBuffer chan []byte,
	mongoSession *mgo.Session,
	t *account.Translation,
	size, sampleRateHertz int,
	audioType speechpb.RecognitionConfig_AudioEncoding,
	language string,
	model string) Stream {
	client, err := speech.NewClient(ctx)
	if err != nil {
		log.Fatal(err)
	}

	speechStream, err := client.StreamingRecognize(ctx)
	if err != nil {
		log.Fatal(err)
	}

	stream := Stream{
		ctx:             ctx,
		speechClient:    client,
		speechStream:    speechStream,
		fileBuffer:      fileBuffer,
		Closed:          false,
		StreamResp:      make(chan []byte),
		StreamErr:       make(chan []byte),
		mutex:           &sync.Mutex{},
		size:            size,
		mongoSession:    mongoSession,
		translation:     t,
		sampleRateHertz: int32(sampleRateHertz),
		audioType:       audioType,
		language:        language,
		model:           model,
	}

	return stream
}

func (s *Stream) initStream() {
	if err := s.speechStream.Send(&speechpb.StreamingRecognizeRequest{
		StreamingRequest: &speechpb.StreamingRecognizeRequest_StreamingConfig{
			StreamingConfig: &speechpb.StreamingRecognitionConfig{
				Config: &speechpb.RecognitionConfig{
					Encoding:                   s.audioType,
					SampleRateHertz:            s.sampleRateHertz,
					LanguageCode:               s.language,
					EnableAutomaticPunctuation: true,
					UseEnhanced:                true,
					Model:                      s.model,
					DiarizationConfig: &speechpb.SpeakerDiarizationConfig{
						EnableSpeakerDiarization: true,
						MinSpeakerCount:          2,
						MaxSpeakerCount:          3,
					},
				},
			},
		},
	}); err != nil {
		log.Fatal(err)
	}
}

func (s *Stream) startStream() {
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
				s.StreamErr <- []byte(fmt.Sprintf("Could not send audio: %v", err))
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
	s.mutex.Lock()
	s.inputEOF = true
	s.mutex.Unlock()
}

func (s *Stream) listen(done chan bool) {
	streamCp := s.speechStream
	for {
		resp, err := streamCp.Recv()
		if err == io.EOF {
			fmt.Printf("EOF\n")
			break
		}
		if err != nil {
			serialized, _ := json.Marshal(err)
			s.StreamErr <- serialized
			break
		}
		if err := resp.Error; err != nil {
			serialized, _ := json.Marshal(err)
			s.StreamErr <- serialized
			break
		}
		for _, result := range resp.Results {
			serialized, _ := json.Marshal(account.TranscriptFromResult(result))
			sessionCopy := s.mongoSession.Copy()
			collection := sessionCopy.DB("s2t").C("translations")
			query := bson.M{
				"$push": bson.M{
					"transcripts": result,
				},
			}
			collection.UpdateId(s.translation.Id, query)
			sessionCopy.Close()
			if !s.Closed {
				s.StreamResp <- serialized
			}
		}
	}
	done <- true
}

func (s *Stream) Start() {
	s.initStream()
	go s.startStream()
	done := make(chan bool)
	go s.listen(done)
	go func() {
		for {
			duration := 5*time.Minute - 30*time.Second
			time.Sleep(duration)
			s.mutex.Lock()
			if s.inputEOF {
				s.mutex.Unlock()
				break
			}
			_ = s.speechStream.CloseSend()
			speechStream, err := s.speechClient.StreamingRecognize(s.ctx)
			if err != nil {
				log.Fatal(err)
			}
			s.speechStream = speechStream
			s.initStream()
			s.mutex.Unlock()
		}
	}()
	<-done
}
