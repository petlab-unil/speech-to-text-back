package account

import (
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"time"
)

func createAccount(account *Account, mongoSession *mgo.Session) error {
	sessionCopy := mongoSession.Copy()
	defer sessionCopy.Close()
	collection := sessionCopy.DB("s2t").C("accounts")

	bytesHash, err := bcrypt.GenerateFromPassword([]byte(account.Password), 14)

	if err != nil {
		return err
	}

	account.Password = string(bytesHash)

	return collection.Insert(&account)
}

func identifyAccount(queriedAccount *Account, mongoSession *mgo.Session) (bool, error) {
	sessionCopy := mongoSession.Copy()
	defer sessionCopy.Close()
	collection := sessionCopy.DB("s2t").C("accounts")
	var dbAccount *Account
	err := collection.Find(bson.M{"name": queriedAccount.Name}).One(&dbAccount)
	if err != nil {
		return false, err
	}

	if dbAccount == nil {
		return false, nil
	}

	bytesHash, err := bcrypt.GenerateFromPassword([]byte(queriedAccount.Password), 14)

	if err != nil {
		return false, err
	}

	return string(bytesHash) != dbAccount.Password, nil
}

func createSession(account *Account, mongoSession *mgo.Session) (*Session, error) {
	sessionCopy := mongoSession.Copy()
	defer sessionCopy.Close()
	collection := sessionCopy.DB("s2t").C("sessions")

	session := Session{
		user:      account.Id,
		CreatedAt: time.Now(),
	}

	err := collection.Insert(&session)

	if err != nil {
		return nil, err
	}

	var createdSession Session
	err = collection.Find(&session).One(&createdSession)

	if err != nil {
		return nil, err
	}

	return &createdSession, nil
}

type errorString struct {
	s string
}

func (e *errorString) Error() string {
	return e.s
}

func CheckSession(sessionId string, mongoSession *mgo.Session) (bool, error) {
	sessionCopy := mongoSession.Copy()
	defer sessionCopy.Close()
	collection := sessionCopy.DB("s2t").C("sessions")

	if len(sessionId) < 12 {
		return false, &errorString{"Invalid sessionID"}
	}
	oid := bson.ObjectIdHex(sessionId)

	var session *Session
	err := collection.Find(bson.M{"_id": oid}).One(&session)

	if err != nil {
		return false, err
	}

	if session == nil {
		return false, nil
	}

	oneDay := 24 * time.Hour
	return session.CreatedAt.Add(oneDay).Sub(time.Now()) > 0, nil
}
