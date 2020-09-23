package account

import (
	"gopkg.in/mgo.v2/bson"
	"time"
)

type Account struct {
	Id       bson.ObjectId `json:"_id" bson:"_id,omitempty"`
	Name     string        `json:"name"`
	Password string        `json:"password"`
}

type Session struct {
	Id        bson.ObjectId `json:"_id" bson:"_id,omitempty"`
	user      bson.ObjectId
	CreatedAt time.Time `json:"created_at,omitempty" bson:"created_at,omitempty"`
}
