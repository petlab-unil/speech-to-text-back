package account

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

func FindSession(mongoSession *mgo.Session, id string) (*Session, error) {
	sessionCopy := mongoSession.Copy()
	defer sessionCopy.Close()
	collection := sessionCopy.DB("s2t").C("sessions")

	_, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}
	oid := bson.ObjectIdHex(id)

	var session Session
	err = collection.Find(bson.M{"_id": oid}).One(&session)

	if err != nil {
		return nil, err
	}

	return &session, nil
}

func CreateTranslation(mongoSession *mgo.Session, fileName string, oid bson.ObjectId) (*Translation, error) {
	sessionCopy := mongoSession.Copy()
	defer sessionCopy.Close()
	collection := sessionCopy.DB("s2t").C("translations")

	newTranslation := Translation{
		FileName:    fileName,
		Transcripts: make([]Transcript, 0),
	}

	err := collection.Insert(newTranslation)

	if err != nil {
		return nil, err
	}

	err = collection.Find(newTranslation).One(&newTranslation)

	if err != nil {
		return nil, err
	}

	collection = sessionCopy.DB("s2t").C("accounts")
	query := bson.M{
		"$push": bson.M{
			"translations": newTranslation.Id,
		},
	}

	err = collection.UpdateId(oid, query)

	if err != nil {
		return nil, err
	}

	return &newTranslation, nil
}

func FullAccount(mongoSession *mgo.Session, id string) (*bson.M, error) {
	sessionCopy := mongoSession.Copy()
	defer sessionCopy.Close()

	sessOid := bson.ObjectIdHex(id)
	collection := sessionCopy.DB("s2t").C("sessions")
	var sess Session
	err := collection.FindId(sessOid).One(&sess)

	if err != nil {
		return nil, err
	}

	query := []bson.M{
		{
			"$match": bson.M{
				"_id": sess.User,
			},
		},
		{
			"$lookup": bson.M{
				"from":         "translations",
				"localField":   "translations",
				"foreignField": "_id",
				"as":           "translations",
			},
		},
		{
			"$unwind": "$translations",
		},
		{
			"$project": bson.M{
				"password": 0,
			},
		},
	}

	println(sess.User.Hex())

	collection = sessionCopy.DB("s2t").C("accounts")
	var a bson.M
	err = collection.Pipe(query).One(&a)

	if err != nil {
		return nil, err
	}

	return &a, err
}
