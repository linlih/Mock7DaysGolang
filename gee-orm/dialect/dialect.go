package dialect

import "reflect"

var dialectMap = map[string]Dialect{}

type Dialect interface {
	DataTypeOf(typ reflect.Value) string                    // 用于将Go语言类型转换成数据库类型
	TableExistSQL(tableName string) (string, []interface{}) // 返回某个表是否存在SQL语句
}

func RegisterDialect(name string, dialect Dialect) {
	dialectMap[name] = dialect
}

func GetDialect(name string) (dialect Dialect, ok bool) {
	dialect, ok = dialectMap[name]
	return
}
