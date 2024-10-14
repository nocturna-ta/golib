package mongodb

import (
	"encoding/json"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/bson/bsonrw"
	"reflect"
)

var (
	tJSON = reflect.TypeOf(json.RawMessage{})
)

func jsonEncodeValue(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, val reflect.Value) error {
	if !val.IsValid() || val.Type() != tJSON {
		return bsoncodec.ValueEncoderError{Name: "jsonEncodeValue", Types: []reflect.Type{tJSON}, Received: val}
	}
	b := val.Interface().(json.RawMessage)
	return vw.WriteString(string(b))
}

func jsonDecodeValue(dc bsoncodec.DecodeContext, vr bsonrw.ValueReader, val reflect.Value) error {
	if !val.CanSet() || val.Type() != tJSON {
		return bsoncodec.ValueDecoderError{Name: "jsonDecodeValue", Types: []reflect.Type{tJSON}, Received: val}
	}

	var data string
	var err error
	switch vrType := vr.Type(); vrType {
	case bson.TypeString:
		data, err = vr.ReadString()
	case bson.TypeNull:
		err = vr.ReadNull()
	case bson.TypeUndefined:
		err = vr.ReadUndefined()
	default:
		return fmt.Errorf("cannot decode %v into a raw json", vrType)
	}

	if err != nil {
		return err
	}

	val.Set(reflect.ValueOf([]byte(data)))
	return nil
}
