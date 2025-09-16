package zk

import (
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/curtisnewbie/miso/miso"
	"github.com/curtisnewbie/miso/util"
	"github.com/curtisnewbie/miso/util/errs"
	"github.com/go-zookeeper/zk"
)

var module = miso.InitAppModuleFunc(func() *zkModule {
	return &zkModule{
		mut: &sync.RWMutex{},
	}
})

func init() {
	miso.RegisterBootstrapCallback(miso.ComponentBootstrap{
		Name:      "Boostrap Zookeeper",
		Bootstrap: zkBootstrap,
		Condition: zkBootstrapCondition,
		Order:     miso.BootstrapOrderL1,
	})
}

type zkModule struct {
	mut    *sync.RWMutex
	client *zk.Conn
}

func zkBootstrap(rail miso.Rail) error {
	m := module()
	m.mut.Lock()
	defer m.mut.Unlock()
	hosts := miso.GetPropStrSlice(PropZkHost)
	miso.Infof("Connecting to Zookeeper: %+v", hosts)
	c, _, err := zk.Connect(hosts, time.Second*time.Duration(miso.GetPropInt(PropZkSessionTimeout)), func(zc *zk.Conn) {
		zc.SetLogger(miso.EmptyRail())
	})
	if err != nil {
		return errs.WrapErrf(err, "connect zookeeper failed")
	}
	m.client = c
	miso.AddShutdownHook(func() { m.client.Close() })
	return nil
}

func zkBootstrapCondition(rail miso.Rail) (bool, error) {
	return miso.GetPropBool(PropZkEnabled), nil
}

func Conn() *zk.Conn {
	m := module()
	m.mut.RLock()
	defer m.mut.RUnlock()
	return m.client
}

func CreateEphNode(p string, dat []byte) error {
	_, err := Conn().Create(p, dat, zk.FlagEphemeral, zk.WorldACL(zk.PermAll))
	return err
}

func CreatePerNode(p string, dat []byte) error {
	_, err := Conn().Create(p, dat, zk.FlagPersistent, zk.WorldACL(zk.PermAll))
	return err
}

func Watch(p string) (<-chan zk.Event, error) {
	_, _, ch, err := Conn().ExistsW(p)
	return ch, err
}

func Get(p string) ([]byte, error) {
	buf, _, err := Conn().Get(p)
	return buf, err
}

// Create LeaderElection.
//
// For the same rootPath, only one *LeaderElection should be created and used.
func NewLeaderElection(rootPath string) *LeaderElection {
	if !strings.HasPrefix(rootPath, "/") {
		rootPath = "/" + rootPath
	}
	return &LeaderElection{
		mu:         &sync.Mutex{},
		rootPath:   rootPath,
		leaderPath: rootPath + "/leader",
		nodeIdFunc: func() string { return util.GetLocalIPV4() },
	}
}

type LeaderElection struct {
	mu         *sync.Mutex
	rootPath   string
	leaderPath string
	nodeIdFunc func() string
}

// Elect leader.
//
// If current node becomes the leader, leaderDo is called exactly one time.
//
// If current node fails to become the leader, it blocks indefinitively until it becomes the leader.
//
// You can cancel the election, using Rail.WithCancel() or Rail.WithTimeout().
func (l *LeaderElection) Elect(rail miso.Rail, leaderDo func()) (bool, error) {
	return electLeader(rail, l, l.nodeIdFunc(), leaderDo)
}

func electLeader(rail miso.Rail, l *LeaderElection, nodeId string, hook func()) (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if err := CreatePerNode(l.rootPath, nil); err != nil {
		rail.Debugf("Create parent path failed (expected), %v", err)
	}

	err := CreateEphNode(l.leaderPath, util.UnsafeStr2Byt(nodeId))

	// we are the leader
	if err == nil {
		rail.Info("Elected to be the leader, running hook")
		hook()
		return true, nil
	}

	if errors.Is(err, zk.ErrConnectionClosed) {
		return false, errs.WrapErrf(err, "zk connection closed")
	}

	// node occupied, watch and try again
	if errors.Is(err, zk.ErrNodeExists) {
		rail.Info("Waiting to become leader")

		ch, err := Watch(l.leaderPath)
		if err != nil {
			return false, errs.WrapErrf(err, "failed to watch zk path: %v", l.leaderPath)
		}

		rail, cancel := rail.WithCancel()
		miso.AddShutdownHook(func() { cancel() }) // if the server shutdowns, it should stop as well
		c := rail.Context()
		for {
			select {
			case e := <-ch:
				if e.Path == l.leaderPath && e.Type == zk.EventNodeDeleted {
					rail.Infof("received zknode event, %#v", e)

					if err := CreateEphNode(l.leaderPath, util.UnsafeStr2Byt(nodeId)); err != nil {
						rail.Errorf("received EventNodeDeleted but failed to elect leader, %v", err)
						if errors.Is(err, zk.ErrConnectionClosed) {
							return false, errs.WrapErrf(err, "zk connection closed")
						}
					} else {
						rail.Info("Elected to be the leader, running hook")
						hook()
						return true, nil
					}
				}
			case <-c.Done():
				rail.Info("Aborting LeaderElection, context closed")
				return false, nil
			}
		}
	}

	// unknown error
	return false, errs.WrapErrf(err, "create zk node failed")
}
