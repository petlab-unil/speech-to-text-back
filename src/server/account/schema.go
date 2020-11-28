package account

import (
	speechpb "google.golang.org/genproto/googleapis/cloud/speech/v1"
	"gopkg.in/mgo.v2/bson"
	"time"
)

type Word struct {
	StartTime  string `json:"starttime" bson:"starttime"`
	EndTime    string `json:"endtime" bson:"endtime"`
	Word       string `json:"word" bson:"word"`
	SpeakerTag int8   `json:"speakertag" bson:"speakertag"`
}

type Alternative struct {
	Confidence float32 `json:"confidence" bson:"confidence"`
	Transcript string  `json:"transcript" bson:"transcript"`
	Words      []Word  `json:"words" bson:"words"`
}

type ResultEndTime struct {
	Nanos   int32 `json:"nanos" bson:"nanos"`
	Seconds int64 `json:"seconds" bson:"seconds"`
}

type Transcript struct {
	Alternatives  []Alternative `json:"alternatives" bson:"alternatives"`
	IsFinal       bool          `json:"isfinal" bson:"isfinal"`
	ResultEndTime ResultEndTime `json:"resultendtime" bson:"resultendtime"`
}

func TranscriptFromResult(result *speechpb.StreamingRecognitionResult) Transcript {
	alternatives := make([]Alternative, len(result.Alternatives))
	for i, alt := range result.Alternatives {
		alternatives[i] = Alternative{
			Confidence: alt.Confidence,
			Transcript: alt.Transcript,
		}
	}
	return Transcript{
		Alternatives: alternatives,
		IsFinal:      result.IsFinal,
		ResultEndTime: ResultEndTime{
			Nanos:   result.ResultEndTime.Nanos,
			Seconds: result.ResultEndTime.Seconds,
		},
	}
}

type Translation struct {
	Id          bson.ObjectId `json:"_id" bson:"_id,omitempty"`
	FileName    string        `json:"file_name" bson:"file_name"`
	Transcripts []Transcript  `json:"transcripts" bson:"transcripts"`
}

type Account struct {
	Id           bson.ObjectId `json:"_id" bson:"_id,omitempty"`
	Name         string        `json:"name" bson:"name"`
	Password     string        `json:"password" bson:"password"`
	Translations []Translation `json:"translations" bson:"translations"`
}

type Session struct {
	Id        bson.ObjectId `json:"_id" bson:"_id,omitempty"`
	User      bson.ObjectId
	CreatedAt time.Time `json:"created_at,omitempty" bson:"created_at,omitempty"`
}
