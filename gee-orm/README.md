# Gee-ORM

ORM(Object Relational Mapping)是描述对象和数据库之间的映射的元数据，将面向对象语言程序中的对象自动持久化到数据库中。

目录结构：
- cmd_test 测试代码
- dialect 中文意思是方言，在数据库中，MySQL是一个方言，Oracle也是为一个方言，所以对于特定的数据库类型就放在这里面，比如这里的sqlite，每个方言需要共通的含义，也就是每个方言都需要实现dialect.go的接口。
- log 日志输出相关
- schema 分析高级语言的对象结构，转换成数据库的类型
- session 表示对数据库的操作，连接到数据库的一个会话操作


