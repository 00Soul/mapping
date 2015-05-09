package json

import (
	"encoding/json"
	"github.com/00Soul/mappings"
	"reflect"
)

type Serializer struct {
	context *mappings.Context
}

type Encoder struct {
}

type Decoder struct {
}

var defaultSerializer *Serializer

func New(c *mappings.Context) *Serializer {
	s := new(Serializer)
	s.Use(c)

	return s
}

func (s *Serializer) Use(c *mappings.Context) {
	s.context = c
}

func getDefault() *Serializer {
	if defaultSerializer == nil {
		defaultSerializer = New(mappings.Global())
	}

	return defaultSerializer
}

func Use(c *mappings.Context) {
	getDefault().Use(c)
}

func Marshal(i interface{}) ([]byte, error) {
	return getDefault().Marshal(i)
}

func Unmarshal(data []byte, i interface{}) error {
	return getDefault().Unmarshal(data, i)
}

func NewEncoder(w io.Writer) *Encoder {
	return NewEncoderWithContext(w, nil)
}

func NewEncoderWithContext(w io.Writer, c mappings.Context) *Encoder {
	encoder := new(Encoder)
	encoder.writer = w
	encoder.context = c

	return encoder
}

func (e *Encoder) Encode(i interface{}) error {
	if e.context == nil {
		e.context = mappings.Global()
	}

	_, err := e.Write(s.flatten(i))

	return err
}

func (s Serializer) Marshal(i interface{}) ([]byte, error) {
	return json.Marshal(s.flatten(i))
}

func (s Serializer) flatten(i interface{}) interface{} {
	var v interface{}

	t := reflect.TypeOf(i)

	switch t.Kind() {
	case reflect.Struct:
		v = s.toMap(i)
	case reflect.Slice:
		v = s.toSlice(i)
	default:
		if mapping := s.context.Get(t); mapping != nil {
			if fn := tm.GetFlattenFunc(); fn != nil {
				v = fn(i)
			} else {
				v = i
			}
		} else {
			v = i
		}
	}

	return v
}

func (s Serializer) Unmarshal(data []byte, i interface{}) error {
	var v interface{}

	err := json.Unmarshal(data, &v)
	obj := s.unflatten(v, reflect.TypeOf(i))
	i = obj

	return err
}

func (s Serializer) unflatten(i interface{}, t reflect.Type) interface{} {
	v := reflect.New(t).Elem().Interface()

	mapping := s.context.Get(t)
	failed := false

	if mapping != nil {
		if fn := tm.GetUnflattenFunc(); fn != nil {
			v = fn(i)
		} else if m, isMap := i.(map[string]interface{}); isMap && t.Kind() == reflect.Struct {
			v = s.fromMap(m, t)
		} else {
			failed = true
		}
	}

	if failed {
		if slice, isSlice := i.([]interface{}); isSlice && t.Kind() == reflect.Slice {
			v = s.fromSlice(slice, t)
		} else {
			if bytes, isByteSlice := i.([]byte); isByteSlice {
				json.Unmarshal(bytes, &v)
			}
		}
	}

	return v
}

func (s Serializer) toSlice(i interface{}) []interface{} {
	slice := make([]interface{}, 0, 1)

	islice, ok := i.([]interface{})
	if ok {
		for _, item := range islice {
			bytes, err := s.Marshal(item)
			if err == nil {
				slice = append(slice, bytes)
			}
		}
	}

	return slice
}

func (s Serializer) fromSlice(i []interface{}, t reflect.Type) interface{} {
	slice := reflect.MakeSlice(reflect.SliceOf(t), 0, 1)

	for _, item := range i {
		slice = reflect.Append(slice, reflect.ValueOf(s.unflatten(item, t)))
	}

	return slice.Interface()
}

func (s Serializer) toMap(i interface{}) map[string][]byte {
	m := make(map[string][]byte)

	v := reflect.ValueOf(i)
	mapping := s.context.Get(v.Type())

	for n := 0; n < v.NumField(); n++ {
		var key string = v.Type().Field(n).Name
		var field *mappings.Field = nil

		if mapping != nil {
			if field = tm.FieldByName(key); field != nil {
				key = field.GetName()
			}
		}

		usedFieldToFlatten := false
		fv := v.Field(n).Interface()

		if field != nil {
			if fn := field.GetFlattenFunc(); fn != nil {
				m[key], _ = json.Marshal(fn(fv))
				usedFieldToFlatten = true
			}
		}

		if !usedFieldToFlatten {
			m[key], _ = s.Marshal(fv)
		}
	}

	return m
}

func (s Serializer) fromMap(m map[string]interface{}, t reflect.Type) reflect.Value {
	p := reflect.New(t)
	v := p.Elem()

	mapping := s.context.Get(v.Type())

	for n := 0; n < v.NumField(); n++ {
		var key string = v.Type().Field(n).Name
		var field *mappings.Field = nil

		if mapping != nil {
			if field = tm.FieldByName(key); field != nil {
				key = field.GetName()
			}
		}

		if value, keyFound := m[key]; keyFound {
			var usedFieldToUnflatten bool = false
			var i interface{}

			if field != nil {
				if fn := field.GetUnflattenFunc(); fn != nil {
					i = fn(value)
					usedFieldToUnflatten = true
				}
			}

			if !usedFieldToUnflatten {
				i = s.unflatten(value, v.Field(n).Type())
			}

			v.Field(n).Set(reflect.ValueOf(i))
		}
	}

	return p
}
