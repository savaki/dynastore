package dynastore

import (
	"reflect"
	"testing"

	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
)

func TestSerializers(t *testing.T) {
	hashKey := securecookie.GenerateRandomKey(64)
	blockKey := securecookie.GenerateRandomKey(32)
	codec := securecookie.New(hashKey, blockKey)
	name := "blah"

	testCases := map[string]struct {
		serializer serializer
	}{
		"secure": {
			serializer: &codecSerializer{codecs: []securecookie.Codec{codec}},
		},
		"plainText": {
			serializer: &gobSerializer{},
		},
	}

	for label, tc := range testCases {
		t.Run(label, func(t *testing.T) {
			s := tc.serializer

			options := &sessions.Options{
				Path:     "path",
				Domain:   "domain",
				MaxAge:   123,
				Secure:   true,
				HttpOnly: true,
			}
			session := &sessions.Session{
				Values: map[interface{}]interface{}{
					"hello": "world",
				},
				Options: options,
			}
			av, err := s.marshal(name, session)
			if err != nil {
				t.Errorf("expected nil; got %v", err)
				return
			}

			restored := &sessions.Session{}
			err = s.unmarshal(name, av, restored)
			if err != nil {
				t.Errorf("expected nil; got %v", err)
				return
			}

			if session.Values["hello"] != "world" {
				t.Errorf("expected hello:world; got %#v\n", session.Values)
				return
			}

			if !reflect.DeepEqual(options, session.Options) {
				t.Errorf("expected %#v; got %#v", options, session.Options)
				return
			}
		})
	}
}
