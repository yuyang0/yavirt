package configs

import (
	"crypto/tls"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "embed"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/pkg/transport"

	"github.com/dustin/go-humanize"
	"github.com/projecteru2/yavirt/pkg/errors"
	"github.com/projecteru2/yavirt/pkg/log"
	"github.com/projecteru2/yavirt/pkg/netx"
	"github.com/urfave/cli/v2"
)

var (
	//go:embed default-config.toml
	DefaultTemplate string
	Conf            = newDefault()
)

type sizeType int64
type subnetType int64

type storageType int64

const (
	localStorage = iota
	cephStorage
)

func (a *sizeType) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		return nil
	}
	var err error
	i, err := humanize.ParseBytes(string(text))
	if err != nil {
		return err
	}
	*a = sizeType(i)
	return nil
}

func (a *subnetType) UnmarshalText(text []byte) error {
	if len(text) < 1 {
		return nil
	}

	dec, err := netx.IPv4ToInt(string(text))
	if err != nil {
		return err
	}
	*a = subnetType(dec)
	return nil
}

func (a *storageType) UnmarshalText(text []byte) (err error) {
	switch strings.ToLower(string(text)) {
	case "local":
		*a = localStorage
	case "ceph":
		*a = cephStorage
	default:
		err = errors.New("invalid storage type")
	}
	return
}
func (a *storageType) IsCeph() bool {
	return *a == cephStorage
}

func (a *storageType) IsLocal() bool {
	return *a == localStorage
}

type CoreConfig struct {
	Addrs               []string `toml:"addrs"`
	Username            string   `toml:"username"`
	Password            string   `toml:"password"`
	StatusCheckInterval Duration `toml:"status_check_interval"`
	NodeStatusTTL       Duration `toml:"nodestatus_ttl"`
	Nodename            string   `toml:"nodename"`
}

type HostConfig struct {
	Name        string     `json:"name" toml:"name"`
	Addr        string     `json:"addr" toml:"addr"`
	Type        string     `json:"type" toml:"type"`
	Subnet      subnetType `json:"subnet" toml:"subnet"`
	CPU         int        `json:"cpu" toml:"cpu"`
	Memory      sizeType   `json:"memory" toml:"memory"`
	Storage     sizeType   `json:"storage" toml:"storage"`
	NetworkMode string     `json:"network,omitempty" toml:"network"`
}

type ETCDConfig struct {
	Prefix    string   `toml:"prefix"`
	Endpoints []string `toml:"endpoints"`
	Username  string   `toml:"username"`
	Password  string   `toml:"password"`
	CA        string   `toml:"ca"`
	Key       string   `toml:"key"`
	Cert      string   `toml:"cert"`
}

type CalicoCnfig struct {
	ConfigFile  string   `toml:"config_file"`
	PoolNames   []string `toml:"pools"`
	GatewayName string   `toml:"gateway"`
	ETCDEnv     string   `toml:"etcd_env"`
}

type CNIConfig struct {
	PluginPath string `toml:"plugin_path"`
	ConfigPath string `toml:"config_path"`
}

type NetworkConfig struct {
	IFNamePrefix string `toml:"ifname_prefix"`
}

type CephConfig struct {
	MonitorAddrs []string `toml:"monitor_addrs"`
	Username     string   `toml:"username"`
	SecretUUID   string   `toml:"secret_uuid"`
}

type LocalConfig struct {
	Dir string `toml:"dir"`
}
type StorageConfig struct {
	Type            storageType `toml:"type"`
	InitGuestVolume bool        `toml:"init_guest_volume"`
	Ceph            CephConfig  `toml:"ceph"`
	Local           LocalConfig `toml:"local"`
}

// Config .
type Config struct {
	Env string `toml:"env"`
	// host-related config
	Host    HostConfig    `toml:"host"`
	Core    CoreConfig    `toml:"core"`
	Etcd    ETCDConfig    `toml:"etcd"`
	Calico  CalicoCnfig   `toml:"calico"`
	CNI     CNIConfig     `toml:"cni"`
	Network NetworkConfig `toml:"network"`
	Storage StorageConfig `toml:"storage"`

	ProfHTTPPort           int      `toml:"prof_http_port"`
	BindHTTPAddr           string   `toml:"bind_http_addr"`
	BindGRPCAddr           string   `toml:"bind_grpc_addr"`
	SkipGuestReportRegexps []string `toml:"skip_guest_report_regexps"`
	EnabledCalicoCNI       bool     `toml:"enabled_calico_cni"`

	VirtTimeout        Duration `toml:"virt_timeout"`
	GracefulTimeout    Duration `toml:"graceful_timeout"`
	HealthCheckTimeout Duration `toml:"health_check_timeout"`
	QMPConnectTimeout  Duration `toml:"qmp_connect_timeout"`

	ImageHubDomain    string `toml:"image_hub_domain"`
	ImageHubNamespace string `toml:"image_hub_namespace"`

	GADiskTimeout Duration `toml:"ga_disk_timeout"`
	GABootTimeout Duration `toml:"ga_boot_timeout"`

	ResizeVolumeMinRatio float64 `toml:"resize_volume_min_ratio"`
	ResizeVolumeMinSize  int64   `toml:"resize_volume_min_size"`

	MinCPU                int   `toml:"min_cpu"`
	MaxCPU                int   `toml:"max_cpu"`
	MinMemory             int64 `toml:"min_memory"`
	MaxMemory             int64 `toml:"max_memory"`
	MinVolumeCap          int64 `toml:"min_volume"`
	MaxVolumeCap          int64 `toml:"max_volume"`
	MaxVolumesCount       int   `toml:"max_volumes_count"`
	MaxSnapshotsCount     int   `toml:"max_snapshots_count"`
	SnapshotRestorableDay int   `toml:"snapshot_restorable_days"`

	MetaTimeout Duration `toml:"meta_timeout"`
	MetaType    string   `toml:"meta_type"`

	VirtDir                 string `toml:"virt_dir"`
	VirtFlockDir            string `toml:"virt_flock_dir"`
	VirtTmplDir             string `toml:"virt_temp_dir"`
	VirtSockDir             string `toml:"virt_sock_dir"`
	VirtBridge              string `toml:"virt_bridge"`
	VirtCPUCachePassthrough bool   `toml:"virt_cpu_cache_passthrough"`

	LogLevel  string `toml:"log_level"`
	LogFile   string `toml:"log_file"`
	LogSentry string `toml:"log_sentry"`

	Batches []*Batch `toml:"batches"`

	// system recovery
	RecoveryOn            bool     `toml:"recovery_on"`
	RecoveryMaxRetries    int      `toml:"recovery_max_retries"`
	RecoveryRetryInterval Duration `toml:"recovery_retry_interval"`
	RecoveryInterval      Duration `toml:"recovery_interval"`
}

func Hostname() string {
	return Conf.Host.Name
}

func newDefault() Config {
	var conf Config
	if err := Decode(DefaultTemplate, &conf); err != nil {
		log.FatalStack(err)
	}

	return conf
}

// Dump .
func (cfg *Config) Dump() (string, error) {
	return Encode(cfg)
}

// Load .
func (cfg *Config) Load(files []string) error {
	for _, path := range files {
		if err := DecodeFile(path, cfg); err != nil {
			return errors.Trace(err)
		}
	}
	return nil
}

func (cfg *Config) Prepare(c *cli.Context) (err error) {
	// try to initialize Hostname
	if c.String("addr") != "" {
		cfg.Host.Addr = c.String("addr")
	}
	if c.String("hostname") != "" {
		cfg.Host.Name = c.String("hostname")
	} else if cfg.Host.Name == "" {
		cfg.Host.Name, err = os.Hostname()
		if err != nil {
			return err
		}
	}

	if cfg.Host.Name == "" {
		cfg.Host.Name = strings.ReplaceAll(cfg.Host.Addr, ".", "-")
	}

	if c.String("log-level") != "" {
		cfg.LogLevel = c.String("log-level")
	}

	if len(c.StringSlice("core-addrs")) > 0 {
		cfg.Core.Addrs = c.StringSlice("core-addrs")
	}
	if c.String("core-username") != "" {
		cfg.Core.Username = c.String("core-username")
	}
	if c.String("core-password") != "" {
		cfg.Core.Password = c.String("core-password")
	}
	// prepare ETCD_ENDPOINTS(Calico needs this environment variable)
	if len(cfg.Etcd.Endpoints) > 0 {
		if err = os.Setenv("ETCD_ENDPOINTS", strings.Join(cfg.Etcd.Endpoints, ",")); err != nil {
			return err
		}
	}

	if cfg.Host.Addr == "" {
		return errors.New("Address must be provided")
	}
	// validate config values
	if cfg.Host.Name == "" {
		return errors.New("Hostname must be provided")
	}
	if len(cfg.Core.Addrs) == 0 {
		return errors.New("Core addresses are needed")
	}
	// network
	if cfg.Host.NetworkMode == "calico" {
		cfg.Network.IFNamePrefix = "cali"
	}
	return cfg.loadVirtDirs()
}

func (cfg *Config) loadVirtDirs() error {
	cfg.VirtFlockDir = filepath.Join(cfg.VirtDir, "flock")
	cfg.VirtTmplDir = filepath.Join(cfg.VirtDir, "template")
	cfg.VirtSockDir = filepath.Join(cfg.VirtDir, "sock")
	// ensure directories
	for _, d := range []string{cfg.VirtFlockDir, cfg.VirtTmplDir, cfg.VirtSockDir} {
		err := os.MkdirAll(d, 0755)
		if err != nil && !os.IsExist(err) {
			return err
		}
	}
	return nil
}

// NewEtcdConfig .
func (cfg *Config) NewEtcdConfig() (etcdcnf clientv3.Config, err error) {
	etcdcnf.Endpoints = cfg.Etcd.Endpoints
	etcdcnf.Username = cfg.Etcd.Username
	etcdcnf.Password = cfg.Etcd.Password
	etcdcnf.TLS, err = cfg.newEtcdTLSConfig()
	return
}

func (cfg *Config) newEtcdTLSConfig() (*tls.Config, error) {
	if len(cfg.Etcd.CA) < 1 || len(cfg.Etcd.Key) < 1 || len(cfg.Etcd.Cert) < 1 {
		return nil, nil //nolint
	}

	return transport.TLSInfo{
		TrustedCAFile: cfg.Etcd.CA,
		KeyFile:       cfg.Etcd.Key,
		CertFile:      cfg.Etcd.Cert,
	}.ClientConfig()
}

// CoreGuestStatusTTL .
func (cfg *Config) CoreGuestStatusTTL() time.Duration {
	return 3 * cfg.Core.StatusCheckInterval.Duration() //nolint:gomnd // TTL is 3 times the interval
}

// CoreGuestStatusCheckInterval .
func (cfg *Config) CoreGuestStatusCheckInterval() time.Duration {
	return cfg.Core.StatusCheckInterval.Duration()
}

// CoreGRPCTimeout .
func (cfg *Config) CoreGRPCTimeout() time.Duration {
	return cfg.CoreStatusReportInterval() / 3 //nolint:gomnd // report timeout 3 times per interval
}

// CoreStatusReportInterval .
func (cfg *Config) CoreStatusReportInterval() time.Duration {
	return cfg.Core.StatusCheckInterval.Duration() / 3 //nolint:gomnd // report 3 times every check
}

// HasImageHub indicates whether the config has ImageHub configurations.
func (cfg *Config) HasImageHub() bool {
	return len(cfg.ImageHubDomain) > 0 && len(cfg.ImageHubNamespace) > 0
}
