// Copyright 2012, Google Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package proto

// DO NOT EDIT.
// FILE GENERATED BY BSONGEN.

import (
	"bytes"

	"github.com/youtube/vitess/go/bson"
	"github.com/youtube/vitess/go/bytes2"
)

// MarshalBson bson-encodes Query.
func (query *Query) MarshalBson(buf *bytes2.ChunkedWriter, key string) {
	bson.EncodeOptionalPrefix(buf, bson.Object, key)
	lenWriter := bson.NewLenWriter(buf)

	bson.EncodeString(buf, "Sql", query.Sql)
	// map[string]interface{}
	{
		bson.EncodePrefix(buf, bson.Object, "BindVariables")
		lenWriter := bson.NewLenWriter(buf)
		for _k, _v1 := range query.BindVariables {
			bson.EncodeInterface(buf, _k, _v1)
		}
		lenWriter.Close()
	}
	query.TabletType.MarshalBson(buf, "TabletType")
	// *Session
	if query.Session == nil {
		bson.EncodePrefix(buf, bson.Null, "Session")
	} else {
		(*query.Session).MarshalBson(buf, "Session")
	}
	bson.EncodeBool(buf, "NotInTransaction", query.NotInTransaction)

	lenWriter.Close()
}

// UnmarshalBson bson-decodes into Query.
func (query *Query) UnmarshalBson(buf *bytes.Buffer, kind byte) {
	switch kind {
	case bson.EOO, bson.Object:
		// valid
	case bson.Null:
		return
	default:
		panic(bson.NewBsonError("unexpected kind %v for Query", kind))
	}
	bson.Next(buf, 4)

	for kind := bson.NextByte(buf); kind != bson.EOO; kind = bson.NextByte(buf) {
		switch bson.ReadCString(buf) {
		case "Sql":
			query.Sql = bson.DecodeString(buf, kind)
		case "BindVariables":
			// map[string]interface{}
			if kind != bson.Null {
				if kind != bson.Object {
					panic(bson.NewBsonError("unexpected kind %v for query.BindVariables", kind))
				}
				bson.Next(buf, 4)
				query.BindVariables = make(map[string]interface{})
				for kind := bson.NextByte(buf); kind != bson.EOO; kind = bson.NextByte(buf) {
					_k := bson.ReadCString(buf)
					var _v1 interface{}
					_v1 = bson.DecodeInterface(buf, kind)
					query.BindVariables[_k] = _v1
				}
			}
		case "TabletType":
			query.TabletType.UnmarshalBson(buf, kind)
		case "Session":
			// *Session
			if kind != bson.Null {
				query.Session = new(Session)
				(*query.Session).UnmarshalBson(buf, kind)
			}
		case "NotInTransaction":
			query.NotInTransaction = bson.DecodeBool(buf, kind)
		default:
			bson.Skip(buf, kind)
		}
	}
}
