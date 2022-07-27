package nacos

import (
	"github.com/go-admin-team/go-admin-core/config/source"
	"github.com/go-admin-team/go-admin-core/tools/naming"
	"time"
	"unsafe"
)

type Nacos struct {
	opts      source.Options
	host      string
	port      int64
	nameSpace string
	group     string
	dataId    string
}

func (f *Nacos) Read() (*source.ChangeSet, error) {
	newNaming, _ := naming.NewNaming(f.host, f.port, f.nameSpace)
	getConfig, _ := newNaming.GetConfig(f.group, f.dataId)
	cs := &source.ChangeSet{
		Format:    "yaml",
		Source:    getConfig,
		Timestamp: time.Now(),
		Data:      stringToBytes(getConfig),
	}
	cs.Checksum = cs.Sum()
	return cs, nil
}

func stringToBytes(s string) []byte {
	return *(*[]byte)(unsafe.Pointer(
		&struct {
			string
			Cap int
		}{s, len(s)},
	))
}

func (f *Nacos) String() string {
	return "nacos"
}

func (f *Nacos) Watch() (source.Watcher, error) {
	return newWatcher()
}

func (f *Nacos) Write(cs *source.ChangeSet) error {
	return nil
}

func NewSource(host string, port int64, nameSpace, group, dataId string) source.Source {
	return &Nacos{host: host, port: port, nameSpace: nameSpace, group: group, dataId: dataId}
}
