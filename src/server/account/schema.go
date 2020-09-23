package account

import (
	"gopkg.in/mgo.v2/bson"
	"time"
)

type Alternative struct {
	Confidence float64 `json:"confidence" bson:"confidence"`
	Transcript string  `json:"transcript" bson:"transcript"`
}

type ResultEndTime struct {
	Nanos   float64 `json:"nanos" bson:"nanos"`
	Seconds float64 `json:"seconds" bson:"seconds"`
}

type Transcript struct {
	Alternatives  []Alternative `json:"alternatives" bson:"alternatives"`
	IsFinal       bool          `json:"is_final" bson:"is_final"`
	ResultEndTime ResultEndTime `json:"result_end_time" bson:"result_end_time"`
}

type Translation struct {
	Id          bson.ObjectId `json:"_id" bson:"_id,omitempty"`
	FileName    string        `json:"file_name" bson:"file_name"`
	Transcripts []Transcript  `json:"transcripts" bson:"transcripts"`
}

type Account struct {
	Id           bson.ObjectId `json:"_id" bson:"_id,omitempty"`
	Name         string        `json:"name"`
	Password     string        `json:"password"`
	Translations []Translation `json:"translations" bson:"translations"`
}

type Session struct {
	Id        bson.ObjectId `json:"_id" bson:"_id,omitempty"`
	User      bson.ObjectId
	CreatedAt time.Time `json:"created_at,omitempty" bson:"created_at,omitempty"`
}
