package main

import (
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	"infra/chromium/util"
)

func jsonToStruct(json string) *structpb.Struct {
	s := &structpb.Struct{}
	util.PanicOnError(protojson.Unmarshal([]byte(json), s))
	return s
}

func copyPropertiesStruct(properties *structpb.Struct) *structpb.Struct {
	copiedProps := &structpb.Struct{Fields: map[string]*structpb.Value{}}
	for k, v := range properties.GetFields() {
		copiedProps.GetFields()[k] = v
	}
	return copiedProps
}
