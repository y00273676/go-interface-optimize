package after

import (
	"reflect"
	"unsafe"
)

var (
	offset1 uintptr
	offset2 uintptr
	offset3 uintptr
	p       People
	t       = reflect.TypeOf(p)
)

func init() {
	offset1 = t.Field(1).Offset
	offset2 = t.Field(2).Offset
	offset3 = t.Field(3).Offset
}

type People struct {
	Age   int
	Name  string
	Test1 string
	Test2 string
}

type emptyInterface struct {
	typ  *struct{}
	word unsafe.Pointer
}

func New() *People {
	return &People{
		Age:  18,
		Name: "shiina",
		Test1: "test1",
		Test2: "test2",
	}
}

func NewUseReflect() interface{} {
	v := reflect.New(t)

	v.Elem().Field(0).Set(reflect.ValueOf(18))
	v.Elem().Field(1).Set(reflect.ValueOf("shiina"))
	v.Elem().Field(2).Set(reflect.ValueOf("test1"))
	v.Elem().Field(3).Set(reflect.ValueOf("test2"))
	return v.Interface()
}

func NewQuickReflect() interface{} {
	v := reflect.New(t)

	p := v.Interface()
	ptr0 := uintptr((*emptyInterface)(unsafe.Pointer(&p)).word)
	ptr1 := ptr0 + offset1
	ptr2 := ptr0 + offset2
	ptr3 := ptr0 + offset3
	*((*int)(unsafe.Pointer(ptr0))) = 18
	*((*string)(unsafe.Pointer(ptr1))) = "shiina"
	*((*string)(unsafe.Pointer(ptr2))) = "test1"
	*((*string)(unsafe.Pointer(ptr3))) = "test2"
	return p
}

