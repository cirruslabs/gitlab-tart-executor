package tart

type TartConfig struct {
	SSHUsername string
	SSHPassword string
	Memory      uint64
	CPU         uint64
	Softnet     bool
	Headless    bool
	AlwaysPull  bool
}
