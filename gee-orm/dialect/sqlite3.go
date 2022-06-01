package dialect

import (
	"fmt"
	"reflect"
	"time"
)

type sqlite3 struct {
}

func (s sqlite3) DataTypeOf(typ reflect.Value) string {
	switch typ.Kind() {
	case reflect.Bool:
		return "bool"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Uint,
		reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uintptr:
		return "integer"
	case reflect.Uint64, reflect.Int64:
		return "bigint"
	case reflect.Float32, reflect.Float64:
		return "real"
	case reflect.Array, reflect.Slice:
		return "blob"
	case reflect.String:
		return "text"
	case reflect.Struct:
		if _, ok := typ.Interface().(time.Time); ok {
			return "datetime"
		}
	}
	panic(fmt.Sprintf("invalid sql type %s (%s)", typ.Type().Name(), typ.Kind()))
}

func (s sqlite3) TableExistSQL(tableName string) (string, []interface{}) {
	args := []interface{}{tableName}
	return "SELECT name FROM sqlite_master WHERE type='table' and name = ?", args
}

var _ Dialect = (*sqlite3)(nil) // 这样可以确保sqlite3实现了Dialect接口，如果没有实现在编译的时候会报错

func init() {
	RegisterDialect("sqlite3", &sqlite3{})
}
