package discovery

import (
	"net"

	"github.com/hashicorp/serf/serf"
	"go.uber.org/zap"
)

type Config struct {
	NodeName       string
	BindAddr       string
	Tags           map[string]string
	StartJoinAddrs []string
}

type Handler interface {
	Join(name string, addr string) error
	Leave(name string) error
}

type Membership struct {
	Config
	handler Handler
	events  chan serf.Event
	serf    *serf.Serf
	logger  *zap.Logger
}

func NewMembership(h Handler, c Config) (*Membership, error) {
	m := &Membership{
		Config:  c,
		handler: h,
		logger:  zap.L().Named("membership"),
	}
	if err := m.setupSerf(); err != nil {
		return nil, err
	}
	return m, nil
}

func (m *Membership) setupSerf() error {
	addr, err := net.ResolveTCPAddr("tcp", m.BindAddr)
	if err != nil {
		return err
	}
	// 设置配置选项
	c := serf.DefaultConfig()
	c.Init()
	{
		c.MemberlistConfig.BindAddr = addr.IP.String()
		c.MemberlistConfig.BindPort = addr.Port
		m.events = make(chan serf.Event)
		c.EventCh = m.events
		c.Tags = m.Tags
		c.NodeName = m.NodeName
	}
	// 新建一个 Serf 实例
	m.serf, err = serf.Create(c)
	if err != nil {
		return err
	}
	// 启动事件处理线程
	go m.eventHandler()
	// 加入已经存在的（如果有的话）集群
	if m.StartJoinAddrs != nil {
		if _, err := m.serf.Join(m.StartJoinAddrs, true); err != nil {
			return err
		}
	}
	return nil
}

func (m *Membership) eventHandler() {
	for event := range m.events {
		members := event.(serf.MemberEvent).Members
		for _, member := range members {
			if !m.isLocal(member) {
				switch event.EventType() {
				case serf.EventMemberJoin:
					m.handleJoin(member)
				case serf.EventMemberLeave:
					m.handleLeave(member)
				}
			}
		}
	}
}

func (m *Membership) isLocal(member serf.Member) bool {
	return m.serf.LocalMember().Name == member.Name
}

func (m *Membership) handleJoin(member serf.Member) {
	if err := m.handler.Join(member.Name, member.Tags["rpc_addr"]); err != nil {
		m.logError("failed to join", err, member)
	}
}

func (m *Membership) handleLeave(member serf.Member) {
	if err := m.handler.Leave(member.Name); err != nil {
		m.logError("failed to leave", err, member)
	}
}

func (m *Membership) logError(msg string, err error, member serf.Member) {
	m.logger.Error(
		msg,
		zap.Error(err),
		zap.String("name", member.Name),
		zap.String("rpc_addr", member.Tags["rpc_addr"]),
	)
}

func (m *Membership) Members() []serf.Member {
	return m.serf.Members()
}

func (m *Membership) Leave() error {
	return m.serf.Leave()
}
