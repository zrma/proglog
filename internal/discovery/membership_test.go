package discovery

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/serf/serf"
	"github.com/stretchr/testify/require"
	"github.com/travisjeffery/go-dynaport"
)

func TestMembership(t *testing.T) {
	f0 := newFixture(t, nil)
	f1 := newFixture(t, f0.members)
	f2 := newFixture(t, f1.members)

	members := f2.members // f0, f1, f2 whole members
	handler := f0.handler // f0's handler

	first := members[0]
	last := members[len(members)-1]

	wantJoinCnt := len(members) - 1

	require.Eventually(t, func() bool {
		return len(handler.joins) == wantJoinCnt &&
			len(first.Members()) == len(members) &&
			len(handler.leaves) == 0
	}, 3*time.Second, 250*time.Millisecond)

	require.NoError(t, last.Leave())

	require.Eventually(t, func() bool {
		return len(handler.joins) == wantJoinCnt &&
			len(members[0].Members()) == len(members) &&
			serf.StatusLeft == first.Members()[2].Status &&
			len(handler.leaves) == 1
	}, 3*time.Second, 250*time.Millisecond)

	require.Equal(t, fmt.Sprintf("member-%d", 2), <-handler.leaves)
}

type fixture struct {
	members []*Membership
	handler *fakeHandler
}

func newFixture(t *testing.T, members []*Membership) *fixture {
	t.Helper()

	id := len(members)
	ports := dynaport.Get(1)
	require.NotEmpty(t, ports)

	addr := fmt.Sprintf("%s:%d", "127.0.0.1", ports[0])
	tags := map[string]string{
		"rpc_addr": addr,
	}

	cfg := Config{
		NodeName: fmt.Sprintf("member-%d", id),
		BindAddr: addr,
		Tags:     tags,
	}

	h := &fakeHandler{}
	if len(members) > 0 {
		cfg.InitialPeers = []string{members[0].Config.BindAddr}
	} else {
		h.joins = make(chan map[string]string, 3)
		h.leaves = make(chan string, 3)
	}

	m, err := New(h, cfg)
	require.NoError(t, err)

	return &fixture{
		members: append(members, m),
		handler: h,
	}
}

type fakeHandler struct {
	joins  chan map[string]string
	leaves chan string
}

func (h *fakeHandler) Join(name, addr string) error {
	if h.joins != nil {
		h.joins <- map[string]string{
			"name": name,
			"addr": addr,
		}
	}
	return nil
}

func (h *fakeHandler) Leave(name string) error {
	if h.leaves != nil {
		h.leaves <- name
	}
	return nil
}
