package config

type Nacos struct {
	Host      string
	Port      int64
	NameSpace string
	Group     string
	DataId    string
}

var NacosConfig = new(Nacos)
