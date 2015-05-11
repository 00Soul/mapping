package json

import (
	"encoding/json"
	"github.com/00Soul/mappings"
	"io"
	"io/ioutil"
	"reflect"
)

type Serializer struct {
	context *mappings.Context
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
	return MarshalWithContext(i, mappings.Global())
}

func MarshalWithContext(i interface{}, c *mappings.Context) ([]byte, error) {
	return json.Marshal(flatten(i, c))
}

func Encode(w io.Writer, i interface{}) error {
	return EncodeWithContext(w, i, mappings.Global())
}

func EncodeWithContext(w io.Writer, i interface{}, c *mappings.Context) error {
	var bytes []byte
	var err error

	if bytes, err = MarshalWithContext(i, c); err == nil {
		_, err = w.Write(bytes)
	}
	/*
		bytes, marshalError := MarshalWithContext(i, c)
		if marshalError != nil {
			err = marshalError
		} else {
			_, writeError := w.Write(bytes)
			if writeError != nil {
				err = writeError
			}
		}
	*/
	return err
}

func flatten(i interface{}, c *mappings.Context) interface{} {
	var v interface{}

	t := reflect.TypeOf(i)

	switch t.Kind() {
	case reflect.Struct:
		v = toMap(i, c)
	case reflect.Slice:
		v = toSlice(i, c)
	default:
		if mapping := c.Get(t); mapping != nil {
			if fn := mapping.GetFlattenFunc(); fn != nil {
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

func Unmarshal(data []byte, i interface{}) error {
	return UnmarshalWithContext(data, i, mappings.Global())
}

func UnmarshalWithContext(data []byte, i interface{}, c *mappings.Context) error {
	var j interface{}
	var err error

	if err = json.Unmarshal(data, &j); err == nil {
		if p := reflect.ValueOf(i); p.Type().Kind() == reflect.Ptr {
			if v := p.Elem(); v.CanSet() {
				object := unflatten(j, v.Type(), c)
				if reflect.TypeOf(object).AssignableTo(v.Type()) {
					v.Set(reflect.ValueOf(object))
				}
			}
		}
	}

	return err
}

func Decode(r io.Reader, i interface{}) error {
	return DecodeWithContext(r, i, mappings.Global())
}

func DecodeWithContext(r io.Reader, i interface{}, c *mappings.Context) error {
	var data []byte
	var err error

	if data, err = ioutil.ReadAll(r); err == nil {
		err = UnmarshalWithContext(data, i, c)
	}

	return err
}

func unflatten(i interface{}, t reflect.Type, c *mappings.Context) interface{} {
	v := reflect.New(t).Elem().Interface()

	mapping := c.Get(t)
	failed := false

	if mapping != nil {
		if fn := mapping.GetUnflattenFunc(); fn != nil {
			v = fn(i)
		} else if m, isMap := i.(map[string]interface{}); isMap && t.Kind() == reflect.Struct {
			v = fromMap(m, t, c)
		} else {
			failed = true
		}
	}

	if failed {
		if slice, isSlice := i.([]interface{}); isSlice && t.Kind() == reflect.Slice {
			v = fromSlice(slice, t, c)
		} else {
			if bytes, isByteSlice := i.([]byte); isByteSlice {
				json.Unmarshal(bytes, &v)
			}
		}
	}

	return v
}

func toSlice(i interface{}, c *mappings.Context) []interface{} {
	slice := make([]interface{}, 0, 1)

	if islice, ok := i.([]interface{}); ok {
		for _, item := range islice {
			if bytes, err := MarshalWithContext(item, c); err == nil {
				slice = append(slice, bytes)
			}
		}
	}

	return slice
}

func fromSlice(i []interface{}, t reflect.Type, c *mappings.Context) interface{} {
	slice := reflect.MakeSlice(reflect.SliceOf(t), 0, 1)

	for _, item := range i {
		slice = reflect.Append(slice, reflect.ValueOf(unflatten(item, t, c)))
	}

	return slice.Interface()
}

func toMap(i interface{}, c *mappings.Context) map[string][]byte {
	m := make(map[string][]byte)

	v := reflect.ValueOf(i)
	mapping := c.Get(v.Type())

	for n := 0; n < v.NumField(); n++ {
		var key string = v.Type().Field(n).Name
		var field *mappings.Field = nil

		if mapping != nil {
			if field = mapping.FieldByName(key); field != nil {
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
			m[key], _ = MarshalWithContext(fv, c)
		}
	}

	return m
}

func fromMap(m map[string]interface{}, t reflect.Type, c *mappings.Context) reflect.Value {
	p := reflect.New(t)
	v := p.Elem()

	mapping := c.Get(v.Type())

	for n := 0; n < v.NumField(); n++ {
		var key string = v.Type().Field(n).Name
		var field *mappings.Field = nil

		if mapping != nil {
			if field = mapping.FieldByName(key); field != nil {
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
				i = unflatten(value, v.Field(n).Type(), c)
			}

			v.Field(n).Set(reflect.ValueOf(i))
		}
	}

	return p
}
