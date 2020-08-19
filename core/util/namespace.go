package util

import "github.com/songxuexian/agollo/core/cons"

// 配置文件有多种格式，例如：properties、xml、yml、yaml、json等。同样Namespace也具有这些格式。在Portal UI中可以看到“application”的Namespace上有一个“properties”标签，表明“application”是properties格式的。
// 如果使用Http接口直接调用时，对应的namespace参数需要传入namespace的名字加上后缀名，如datasources.json。
func GetNamespace(confType, namespace string) string {
	if confType == "" || confType == cons.ConfigType {
		return namespace
	}
	return namespace + "." + confType
}
