package Speech2Text

import (
	speech "cloud.google.com/go/speech/apiv1"
	"cloud.google.com/go/storage"
	"context"
	"encoding/json"
	speechpb "google.golang.org/genproto/googleapis/cloud/speech/v1"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"io"
	"log"
	"os"
	"speech-to-text-back/src/server/account"
	"time"
)

type Stream struct {
	ctx             context.Context
	speechClient    *speech.Client
	fileBuffer      chan []byte
	StreamResp      chan []byte
	StreamErr       chan []byte
	Closed          bool
	size            int
	mongoSession    *mgo.Session
	translation     *account.Translation
	sampleRateHertz int32
	language        string
	audioType       speechpb.RecognitionConfig_AudioEncoding
	model           string
	streamsCount    int8
	inputEOF        bool
	uploadBuffer    []byte
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
	stream := Stream{
		ctx:             ctx,
		speechClient:    client,
		fileBuffer:      fileBuffer,
		Closed:          false,
		StreamResp:      make(chan []byte),
		StreamErr:       make(chan []byte),
		size:            size,
		mongoSession:    mongoSession,
		translation:     t,
		sampleRateHertz: int32(sampleRateHertz),
		audioType:       audioType,
		language:        language,
		model:           model,
		streamsCount:    0,
		inputEOF:        false,
		uploadBuffer:    []byte{},
	}

	return stream
}

func (s *Stream) listenForFile() {
	currentSize := 0
	for currentSize < s.size {
		fileBuffer := <-s.fileBuffer
		s.uploadBuffer = append(s.uploadBuffer, fileBuffer...)
		currentSize += len(fileBuffer)
	}
}

func (s *Stream) uploadFile(bucket, object string) {
	// bucket := "bucket-name"
	// object := "object-name"
	ctx := context.Background()
	client, err := storage.NewClient(ctx)

	if err != nil {
		serialized, _ := json.Marshal(err)
		s.StreamErr <- serialized
		return
	}

	defer client.Close()

	ctx, cancel := context.WithTimeout(ctx, time.Second*600)
	defer cancel()

	// Upload an object with storage.Writer.
	wc := client.Bucket(bucket).Object(object).NewWriter(ctx)
	if _, err = wc.Write(s.uploadBuffer); err != nil {
		serialized, _ := json.Marshal(err)
		s.StreamErr <- serialized
		return
	}
	if err := wc.Close(); err != nil {
		serialized, _ := json.Marshal(err)
		s.StreamErr <- serialized
		return
	}
	println("UPLOADED")
}

func (s *Stream) translate() {
	ctx := context.Background()
	req := &speechpb.LongRunningRecognizeRequest{
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
		Audio: &speechpb.RecognitionAudio{
			AudioSource: &speechpb.RecognitionAudio_Uri{Uri: gcsURI},
		},
	}
	op, err := s.speechClient.LongRunningRecognize(ctx, req)
	if err != nil {
		serialized, _ := json.Marshal(err)
		s.StreamErr <- serialized
		return
	}
	resp, err := op.Wait(ctx)
	if err != nil {
		serialized, _ := json.Marshal(err)
		s.StreamErr <- serialized
		return
	}

	// Print the results.
	for _, result := range resp.Results {
		serialized, _ := json.Marshal(account.TranscriptFromResult(result))
		sessionCopy := s.mongoSession.Copy()
		collection := sessionCopy.DB("s2t").C("translations")
		query := bson.M{
			"$push": bson.M{
				"transcripts": result,
			},
		}
		_ = collection.UpdateId(s.translation.Id, query)
		sessionCopy.Close()
		if !s.Closed {
			s.StreamResp <- serialized
		}
	}
}
