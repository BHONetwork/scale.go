package types

import (
	"fmt"
	"github.com/freehere107/go-scale-codec/utiles"
	"reflect"
	"strings"
)

type ScaleDecoderOption struct {
	SubType   string
	ValueList []string
	Metadata  *MetadataStruct
}

type TypeMapping struct {
	Names []string
	Types []string
}

type IScaleDecoder interface {
	Init(data ScaleBytes, option *ScaleDecoderOption)
	Process()
	buildStruct()
	NextBytes(int) []byte
	GetNextU8() int
	reset()
}

type ScaleDecoder struct {
	Data        ScaleBytes      `json:"-"`
	TypeString  string          `json:"-"`
	SubType     string          `json:"-"`
	Value       interface{}     `json:"-"`
	RawValue    string          `json:"-"`
	TypeMapping *TypeMapping    `json:"-"`
	Metadata    *MetadataStruct `json:"-"`
}

func (s *ScaleDecoder) Init(data ScaleBytes, option *ScaleDecoderOption) {
	if option != nil {
		s.SubType = option.SubType
		s.Metadata = option.Metadata
	}
	s.Data = data
	s.RawValue = ""
	s.Value = nil
	if s.TypeMapping == nil && s.TypeString != "" {
		s.buildStruct()
	}
}

func (s *ScaleDecoder) Process() {}

func (s *ScaleDecoder) NextBytes(length int) []byte {
	data := s.Data.GetNextBytes(length)
	s.RawValue += utiles.BytesToHex(data)
	return data
}

func (s *ScaleDecoder) GetNextU8() int {
	b := s.NextBytes(1)
	return int(b[0])
}

func (s *ScaleDecoder) getNextBool() bool {
	data := s.NextBytes(1)
	return utiles.BytesToHex(data) == "01"
}

func (s *ScaleDecoder) reset() {
	s.Data.Data = []byte{}
	s.Data.Offset = 0
}

func (s *ScaleDecoder) buildStruct() {
	if s.TypeString != "" && string(s.TypeString[0]) == "(" && string(s.TypeString[len(s.TypeString)-1:]) == ")" {

		var names, types []string
		for k, v := range strings.Split(s.TypeString[1:len(s.TypeString)-1], ",") {
			types = append(types, strings.TrimSpace(v))
			names = append(names, fmt.Sprintf("col%d", k+1))
		}

		s.TypeMapping = &TypeMapping{Names: names, Types: types}
	}
}

func (s *ScaleDecoder) ProcessAndUpdateData(typeString string, args ...string) interface{} {
	r := RuntimeType{}

	if typeRegistry == nil {
		r.Reg()
	}
	c, rc, subType := r.decoderClass(typeString)

	if c == nil {
		panic(fmt.Sprintf("not found decoder class %s", typeString))
	}

	// init
	method, exist := c.MethodByName("Init")
	if exist == false {
		panic(fmt.Sprintf("%s not implement init function", typeString))
	}
	option := ScaleDecoderOption{SubType: subType, ValueList: args}
	method.Func.Call([]reflect.Value{rc, reflect.ValueOf(s.Data), reflect.ValueOf(&option)})

	// process
	rc.MethodByName("Process").Call(nil)

	s.Data.Offset = int(rc.Elem().FieldByName("Data").FieldByName("Offset").Int())
	s.Data.Data = rc.Elem().FieldByName("Data").FieldByName("Data").Bytes()

	return rc.Elem().FieldByName("Value").Interface()
}
