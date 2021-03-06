// Copyright 2015, Google Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package testlib

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/youtube/vitess/go/vt/logutil"
	myproto "github.com/youtube/vitess/go/vt/mysqlctl/proto"
	"github.com/youtube/vitess/go/vt/tabletmanager/tmclient"
	"github.com/youtube/vitess/go/vt/topo"
	"github.com/youtube/vitess/go/vt/wrangler"
	"github.com/youtube/vitess/go/vt/zktopo"
	"golang.org/x/net/context"
)

func TestEmergencyReparentShard(t *testing.T) {
	ctx := context.Background()
	ts := zktopo.NewTestServer(t, []string{"cell1", "cell2"})
	wr := wrangler.New(logutil.NewConsoleLogger(), ts, tmclient.NewTabletManagerClient(), time.Second)

	// Create a master, a couple good slaves
	oldMaster := NewFakeTablet(t, wr, "cell1", 0, topo.TYPE_MASTER)
	newMaster := NewFakeTablet(t, wr, "cell1", 1, topo.TYPE_REPLICA)
	goodSlave1 := NewFakeTablet(t, wr, "cell1", 2, topo.TYPE_REPLICA)
	goodSlave2 := NewFakeTablet(t, wr, "cell2", 3, topo.TYPE_REPLICA)

	// new master
	newMaster.FakeMysqlDaemon.ReadOnly = true
	newMaster.FakeMysqlDaemon.CurrentMasterPosition = myproto.ReplicationPosition{
		GTIDSet: myproto.MariadbGTID{
			Domain:   2,
			Server:   123,
			Sequence: 456,
		},
	}
	newMaster.FakeMysqlDaemon.ExpectedExecuteSuperQueryList = []string{
		"CREATE DATABASE IF NOT EXISTS _vt",
		"SUBCREATE TABLE IF NOT EXISTS _vt.reparent_journal",
		"SUBINSERT INTO _vt.reparent_journal (time_created_ns, action_name, master_alias, replication_position) VALUES",
	}
	newMaster.StartActionLoop(t, wr)
	defer newMaster.StopActionLoop(t)

	// old master, will be scrapped
	oldMaster.FakeMysqlDaemon.ReadOnly = false
	oldMaster.StartActionLoop(t, wr)
	defer oldMaster.StopActionLoop(t)

	// good slave 1 is replicating
	goodSlave1.FakeMysqlDaemon.ReadOnly = true
	goodSlave1.FakeMysqlDaemon.CurrentMasterPosition = myproto.ReplicationPosition{
		GTIDSet: myproto.MariadbGTID{
			Domain:   2,
			Server:   123,
			Sequence: 455,
		},
	}
	goodSlave1.FakeMysqlDaemon.CurrentSlaveStatus = &myproto.ReplicationStatus{
		SlaveIORunning:  true,
		SlaveSQLRunning: true,
	}
	goodSlave1.FakeMysqlDaemon.SetMasterCommandsInput = fmt.Sprintf("%v:%v", newMaster.Tablet.Hostname, newMaster.Tablet.Portmap["mysql"])
	goodSlave1.FakeMysqlDaemon.SetMasterCommandsResult = []string{"set master cmd 1"}
	goodSlave1.FakeMysqlDaemon.ExpectedExecuteSuperQueryList = []string{
		"STOP SLAVE",
		"set master cmd 1",
		"START SLAVE",
	}
	goodSlave1.StartActionLoop(t, wr)
	defer goodSlave1.StopActionLoop(t)

	// good slave 2 is not replicating
	goodSlave2.FakeMysqlDaemon.ReadOnly = true
	goodSlave2.FakeMysqlDaemon.CurrentMasterPosition = myproto.ReplicationPosition{
		GTIDSet: myproto.MariadbGTID{
			Domain:   2,
			Server:   123,
			Sequence: 454,
		},
	}
	goodSlave2.FakeMysqlDaemon.SetMasterCommandsInput = fmt.Sprintf("%v:%v", newMaster.Tablet.Hostname, newMaster.Tablet.Portmap["mysql"])
	goodSlave2.FakeMysqlDaemon.SetMasterCommandsResult = []string{"set master cmd 1"}
	goodSlave2.StartActionLoop(t, wr)
	goodSlave2.FakeMysqlDaemon.ExpectedExecuteSuperQueryList = []string{
		"set master cmd 1",
	}
	defer goodSlave2.StopActionLoop(t)

	// run EmergencyReparentShard
	if err := wr.EmergencyReparentShard(ctx, newMaster.Tablet.Keyspace, newMaster.Tablet.Shard, newMaster.Tablet.Alias, 10*time.Second); err != nil {
		t.Fatalf("EmergencyReparentShard failed: %v", err)
	}

	// check what was run
	if err := newMaster.FakeMysqlDaemon.CheckSuperQueryList(); err != nil {
		t.Fatalf("newMaster.FakeMysqlDaemon.CheckSuperQueryList failed: %v", err)
	}
	if err := oldMaster.FakeMysqlDaemon.CheckSuperQueryList(); err != nil {
		t.Fatalf("oldMaster.FakeMysqlDaemon.CheckSuperQueryList failed: %v", err)
	}
	if err := goodSlave1.FakeMysqlDaemon.CheckSuperQueryList(); err != nil {
		t.Fatalf("goodSlave1.FakeMysqlDaemon.CheckSuperQueryList failed: %v", err)
	}
	if err := goodSlave2.FakeMysqlDaemon.CheckSuperQueryList(); err != nil {
		t.Fatalf("goodSlave2.FakeMysqlDaemon.CheckSuperQueryList failed: %v", err)
	}
	if newMaster.FakeMysqlDaemon.ReadOnly {
		t.Errorf("newMaster.FakeMysqlDaemon.ReadOnly set")
	}
	// old master read-only flag doesn't matter, it is scrapped
	if !goodSlave1.FakeMysqlDaemon.ReadOnly {
		t.Errorf("goodSlave1.FakeMysqlDaemon.ReadOnly not set")
	}
	if !goodSlave2.FakeMysqlDaemon.ReadOnly {
		t.Errorf("goodSlave2.FakeMysqlDaemon.ReadOnly not set")
	}

}

// TestEmergencyReparentShardMasterElectNotBest tries to emergency reparent
// to a host that is not the latest in replication position.
func TestEmergencyReparentShardMasterElectNotBest(t *testing.T) {
	ctx := context.Background()
	ts := zktopo.NewTestServer(t, []string{"cell1", "cell2"})
	wr := wrangler.New(logutil.NewConsoleLogger(), ts, tmclient.NewTabletManagerClient(), time.Second)

	// Create a master, a couple good slaves
	oldMaster := NewFakeTablet(t, wr, "cell1", 0, topo.TYPE_MASTER)
	newMaster := NewFakeTablet(t, wr, "cell1", 1, topo.TYPE_REPLICA)
	moreAdvancedSlave := NewFakeTablet(t, wr, "cell1", 2, topo.TYPE_REPLICA)

	// new master
	newMaster.FakeMysqlDaemon.CurrentMasterPosition = myproto.ReplicationPosition{
		GTIDSet: myproto.MariadbGTID{
			Domain:   2,
			Server:   123,
			Sequence: 456,
		},
	}
	newMaster.StartActionLoop(t, wr)
	defer newMaster.StopActionLoop(t)

	// old master, will be scrapped
	oldMaster.StartActionLoop(t, wr)
	defer oldMaster.StopActionLoop(t)

	// more advanced slave
	moreAdvancedSlave.FakeMysqlDaemon.CurrentMasterPosition = myproto.ReplicationPosition{
		GTIDSet: myproto.MariadbGTID{
			Domain:   2,
			Server:   123,
			Sequence: 457,
		},
	}
	moreAdvancedSlave.StartActionLoop(t, wr)
	defer moreAdvancedSlave.StopActionLoop(t)

	// run EmergencyReparentShard
	if err := wr.EmergencyReparentShard(ctx, newMaster.Tablet.Keyspace, newMaster.Tablet.Shard, newMaster.Tablet.Alias, 10*time.Second); err == nil || !strings.Contains(err.Error(), "is more advanced than master elect tablet") {
		t.Fatalf("EmergencyReparentShard returned the wrong error: %v", err)
	}

	// check what was run
	if err := newMaster.FakeMysqlDaemon.CheckSuperQueryList(); err != nil {
		t.Fatalf("newMaster.FakeMysqlDaemon.CheckSuperQueryList failed: %v", err)
	}
	if err := oldMaster.FakeMysqlDaemon.CheckSuperQueryList(); err != nil {
		t.Fatalf("oldMaster.FakeMysqlDaemon.CheckSuperQueryList failed: %v", err)
	}
	if err := moreAdvancedSlave.FakeMysqlDaemon.CheckSuperQueryList(); err != nil {
		t.Fatalf("moreAdvancedSlave.FakeMysqlDaemon.CheckSuperQueryList failed: %v", err)
	}
}
