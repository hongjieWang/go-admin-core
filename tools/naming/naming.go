package naming

import (
	"fmt"
	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/model"
	"github.com/nacos-group/nacos-sdk-go/vo"
	"sync"
)

// Watch 监听结构体
type Watch struct {
	Service   string
	Callback  func([]ServiceRegistration)
	WaitIndex uint64
	Quit      chan struct{}
}

// Naming Naming对象
type Naming struct {
	sync.RWMutex
	cli       naming_client.INamingClient
	configCli config_client.IConfigClient
	watchs    map[string]*Watch
}

// NewNaming 创建注册中心客户端对象
func NewNaming(url string, port int64, namespaceId string) (Naming, error) {
	// 创建clientConfig的另一种方式
	clientConfig := *constant.NewClientConfig(
		constant.WithNamespaceId(namespaceId), //当namespace是public时，此处填空字符串。
		constant.WithTimeoutMs(5000),
		constant.WithNotLoadCacheAtStart(true),
		constant.WithLogDir("/tmp/nacos/log"),
		constant.WithCacheDir("/tmp/nacos/cache"),
		constant.WithLogLevel("debug"),
	)
	// 创建serverConfig的另一种方式
	serverConfigs := []constant.ServerConfig{
		*constant.NewServerConfig(
			url,
			uint64(port),
			constant.WithScheme("http"),
			constant.WithContextPath("/nacos"),
		),
	}
	namingClient, err := clients.NewNamingClient(
		vo.NacosClientParam{
			ClientConfig:  &clientConfig,
			ServerConfigs: serverConfigs,
		},
	)

	configClient, err := clients.NewConfigClient(vo.NacosClientParam{
		ClientConfig:  &clientConfig,
		ServerConfigs: serverConfigs,
	})

	if err != nil {
		panic(err)
		return Naming{}, err
	}
	return Naming{
		cli:       namingClient,
		configCli: configClient,
		watchs:    make(map[string]*Watch, 1),
	}, nil
}

func (n *Naming) Register(s ServiceRegistration) error {
	reg := vo.RegisterInstanceParam{
		Ip:          s.PublicAddress(),
		Port:        uint64(s.PublicPort()),
		ServiceName: s.ServiceName(),
		Weight:      10,
		Enable:      true,
		Healthy:     true,
		Ephemeral:   true,
		Metadata:    s.GetMeta(),
		GroupName:   s.GroupName(), // 默认值DEFAULT_GROUP
	}
	if reg.Metadata == nil {
		reg.Metadata = make(map[string]string)
	}
	instance, err := n.cli.RegisterInstance(reg)
	if instance {
		fmt.Printf("===server:[%s]====注册成功=====注册IP:[%s]======\n", reg.ServiceName, reg.Ip)
	}
	return err
}

func (n *Naming) Deregister(s ServiceRegistration) error {
	param := vo.DeregisterInstanceParam{
		Ip:          s.PublicAddress(),
		Port:        uint64(s.PublicPort()),
		ServiceName: s.ServiceName(),
		Ephemeral:   true,
		GroupName:   s.GroupName(), // 默认值DEFAULT_GROUP
	}
	instance, err := n.cli.DeregisterInstance(param)
	if instance {
		fmt.Println("服务注销成功")
	}
	return err
}

func (n *Naming) Find(s ServiceRegistration) (ServiceRegistration, error) {
	param := vo.SelectOneHealthInstanceParam{
		ServiceName: s.ServiceName(),
		GroupName:   s.GroupName(), // 默认值DEFAULT_GROUP
	}
	instance, err := n.cli.SelectOneHealthyInstance(param)
	if err != nil {
		return DefaultService{}, err
	}
	return DefaultService{
		Id:   instance.Ip,
		Port: int64(instance.Port),
		Name: instance.ServiceName,
	}, nil
}

func (n *Naming) FindByServerName(serverName, groupName string) (ServiceRegistration, error) {
	param := vo.SelectOneHealthInstanceParam{
		ServiceName: serverName,
		GroupName:   groupName, // 默认值DEFAULT_GROUP
	}
	instance, err := n.cli.SelectOneHealthyInstance(param)
	if err != nil {
		return DefaultService{}, err
	}
	return DefaultService{
		Id:   instance.Ip,
		Port: int64(instance.Port),
		Name: instance.ServiceName,
	}, nil
}

func (n *Naming) FindAll(s ServiceRegistration) ([]ServiceRegistration, error) {
	param := vo.SelectAllInstancesParam{
		ServiceName: s.ServiceName(),
		GroupName:   s.GroupName(), // 默认值DEFAULT_GROUP
	}
	instances, err := n.cli.SelectAllInstances(param)
	services := make([]ServiceRegistration, 0, len(instances))
	if err != nil {
		return services, err
	}
	for _, s := range instances {
		if s.Healthy {
			services = append(services, &DefaultService{
				Name:    s.ServiceName,
				Address: s.Ip,
				Port:    int64(s.Port),
				Meta:    s.Metadata,
			})
		}
	}
	return services, nil
}

// Subscribe 监听服务变化
func (n *Naming) Subscribe(s ServiceRegistration, callback func([]ServiceRegistration)) error {
	param := vo.SubscribeParam{
		ServiceName: s.ServiceName(),
		GroupName:   s.GroupName(), // 默认值DEFAULT_GROUP
		SubscribeCallback: func(services []model.SubscribeService, err error) {
			changeService := make([]ServiceRegistration, 0, len(services))
			for _, s := range services {
				changeService = append(changeService, &DefaultService{
					Name:    s.ServiceName,
					Address: s.Ip,
					Port:    int64(s.Port),
					Meta:    s.Metadata,
				})
			}
			callback(changeService)
		},
	}
	return n.cli.Subscribe(&param)
}

func (n *Naming) GetConfig(group, dataId string) (string, error) {
	return n.configCli.GetConfig(vo.ConfigParam{
		DataId: dataId,
		Group:  group,
		Type:   vo.YAML})
}
