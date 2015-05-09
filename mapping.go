package mapping

import (
	"reflect"
)

type Field struct {
	flattenedName string
	flattenFunc   func(interface{}) interface{}
	unflattenFunc func(interface{}) interface{}
}

type Mapping struct {
	structType    reflect.Type
	flattenFunc   func(interface{}) interface{}
	unflattenFunc func(interface{}) interface{}
	fields        map[string]Field
}

type Context struct {
	mappings map[reflect.Type]Mapping
}

var globalContext *Context = nil

func NewContext() *Context {
	c := new(Context)

	c.mappings = make(map[reflect.Type]Mapping)

	return c
}

func Global() *Context {
	if globalContext == nil {
		globalContext = NewContext()
	}

	return globalContext
}

func New(t reflect.Type) *Mapping {
	return Global().New(t)
}

func Get(t reflect.Type) *Mapping {
	return Global().Get(t)
}

func Del(t reflect.Type) {
	Global().Del(t)
}

func (c *Context) New(t reflect.Type) *Mapping {
	var m Mapping
	m.structType = t
	m.fields = make(map[string]Field)

	c.mappings[t] = m

	return &m
}

func (c Context) Get(t reflect.Type) *Mapping {
	if m, found := c.mappings[t]; found {
		return &m
	} else {
		return nil
	}
}

func (c Context) Del(t reflect.Type) {
	delete(c.mappings, t)
}

func (m Mapping) FieldByName(name string) *Field {
	if f, found := m.fields[name]; found {
		return &f
	} else {
		return nil
	}
}

func (m *Mapping) Field(sf reflect.StructField) *Field {
	var f *Field = m.FieldByName(sf.Name)

	if f == nil {
		tf, found := m.structType.FieldByName(sf.Name)
		if found && tf.Name == sf.Name && tf.PkgPath == sf.PkgPath && tf.Type == sf.Type {
			f = new(Field)
			f.flattenedName = sf.Name

			m.fields[sf.Name] = *f
		}
	}

	return f
}

func (m *Mapping) FlattenFunc(fn func(interface{}) interface{}) *Mapping {
	m.flattenFunc = fn

	return m
}

func (m *Mapping) UnflattenFunc(fn func(interface{}) interface{}) *Mapping {
	m.unflattenFunc = fn

	return m
}

func (m *Mapping) GetFlattenFunc() func(interface{}) interface{} {
	return m.flattenFunc
}

func (m *Mapping) GetUnflattenFunc() func(interface{}) interface{} {
	return m.unflattenFunc
}

func (f *Field) Name(name string) *Field {
	f.flattenedName = name

	return f
}

func (f Field) GetName() string {
	return f.flattenedName
}

func (f *Field) FlattenFunc(fn func(interface{}) interface{}) *Field {
	f.flattenFunc = fn

	return f
}

func (f *Field) UnflattenFunc(fn func(interface{}) interface{}) *Field {
	f.unflattenFunc = fn

	return f
}

func (f *Field) GetFlattenFunc() func(interface{}) interface{} {
	return f.flattenFunc
}

func (f *Field) GetUnflattenFunc() func(interface{}) interface{} {
	return f.unflattenFunc
}
