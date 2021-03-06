// Copyright 2013, Google Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vtctl

import (
	"flag"
	"fmt"
	"time"

	"github.com/youtube/vitess/go/vt/topo"
	"github.com/youtube/vitess/go/vt/wrangler"
	"golang.org/x/net/context"
)

var (
	disableActiveReparents = flag.Bool("disable_active_reparents", false, "if set, do not allow active reparents. Use this to protect a cluster using external reparents.")
)

func init() {
	addCommand("Tablets", command{
		"DemoteMaster",
		commandDemoteMaster,
		"<tablet alias>",
		"Demotes a master tablet."})
	addCommand("Tablets", command{
		"ReparentTablet",
		commandReparentTablet,
		"<tablet alias>",
		"Reparent a tablet to the current master in the shard. This only works if the current slave position matches the last known reparent action."})
	addCommand("Shards", command{
		"InitShardMaster",
		commandInitShardMaster,
		"[-force] [-wait_slave_timeout=<duration>] <keyspace/shard> <tablet alias>",
		"Sets the initial master for a shard. Will make all other tablets in the shard slaves of the provided master. WARNING: this could cause data loss on an already replicating shard, then PlannedReparentShard or EmergencyReparentShard should be used instead."})
	addCommand("Shards", command{
		"PlannedReparentShard",
		commandPlannedReparentShard,
		"<keyspace/shard> <tablet alias>",
		"Reparents the shard to the new master. Both old and new master need to be up and running."})
	addCommand("Shards", command{
		"EmergencyReparentShard",
		commandEmergencyReparentShard,
		"<keyspace/shard> <tablet alias>",
		"Reparents the shard to the new master. Assumes the old master is dead and not responsding."})
}

func commandDemoteMaster(ctx context.Context, wr *wrangler.Wrangler, subFlags *flag.FlagSet, args []string) error {
	if *disableActiveReparents {
		return fmt.Errorf("active reparent actions disable in this cluster")
	}

	if err := subFlags.Parse(args); err != nil {
		return err
	}
	if subFlags.NArg() != 1 {
		return fmt.Errorf("action DemoteMaster requires <tablet alias>")
	}
	tabletAlias, err := topo.ParseTabletAliasString(subFlags.Arg(0))
	if err != nil {
		return err
	}
	tabletInfo, err := wr.TopoServer().GetTablet(tabletAlias)
	if err != nil {
		return err
	}
	_, err = wr.TabletManagerClient().DemoteMaster(ctx, tabletInfo)
	return err
}

func commandReparentTablet(ctx context.Context, wr *wrangler.Wrangler, subFlags *flag.FlagSet, args []string) error {
	if *disableActiveReparents {
		return fmt.Errorf("active reparent actions disable in this cluster")
	}

	if err := subFlags.Parse(args); err != nil {
		return err
	}
	if subFlags.NArg() != 1 {
		return fmt.Errorf("action ReparentTablet requires <tablet alias>")
	}
	tabletAlias, err := topo.ParseTabletAliasString(subFlags.Arg(0))
	if err != nil {
		return err
	}
	return wr.ReparentTablet(ctx, tabletAlias)
}

func commandInitShardMaster(ctx context.Context, wr *wrangler.Wrangler, subFlags *flag.FlagSet, args []string) error {
	if *disableActiveReparents {
		return fmt.Errorf("active reparent actions disable in this cluster")
	}

	force := subFlags.Bool("force", false, "will force the reparent even if the provided tablet is not a master or the shard master")
	waitSlaveTimeout := subFlags.Duration("wait_slave_timeout", 30*time.Second, "time to wait for slaves to catch up in reparenting")
	if err := subFlags.Parse(args); err != nil {
		return err
	}
	if subFlags.NArg() != 2 {
		return fmt.Errorf("action InitShardMaster requires <keyspace/shard> <tablet alias>")
	}
	keyspace, shard, err := topo.ParseKeyspaceShardString(subFlags.Arg(0))
	if err != nil {
		return err
	}
	tabletAlias, err := topo.ParseTabletAliasString(subFlags.Arg(1))
	if err != nil {
		return err
	}
	return wr.InitShardMaster(ctx, keyspace, shard, tabletAlias, *force, *waitSlaveTimeout)
}

func commandPlannedReparentShard(ctx context.Context, wr *wrangler.Wrangler, subFlags *flag.FlagSet, args []string) error {
	if *disableActiveReparents {
		return fmt.Errorf("active reparent actions disable in this cluster")
	}

	waitSlaveTimeout := subFlags.Duration("wait_slave_timeout", 30*time.Second, "time to wait for slaves to catch up in reparenting")
	if err := subFlags.Parse(args); err != nil {
		return err
	}
	if subFlags.NArg() != 2 {
		return fmt.Errorf("action PlannedReparentShard requires <keyspace/shard> <tablet alias>")
	}

	keyspace, shard, err := topo.ParseKeyspaceShardString(subFlags.Arg(0))
	if err != nil {
		return err
	}
	tabletAlias, err := topo.ParseTabletAliasString(subFlags.Arg(1))
	if err != nil {
		return err
	}
	return wr.PlannedReparentShard(ctx, keyspace, shard, tabletAlias, *waitSlaveTimeout)
}

func commandEmergencyReparentShard(ctx context.Context, wr *wrangler.Wrangler, subFlags *flag.FlagSet, args []string) error {
	if *disableActiveReparents {
		return fmt.Errorf("active reparent actions disable in this cluster")
	}

	waitSlaveTimeout := subFlags.Duration("wait_slave_timeout", 30*time.Second, "time to wait for slaves to catch up in reparenting")
	if err := subFlags.Parse(args); err != nil {
		return err
	}
	if subFlags.NArg() != 2 {
		return fmt.Errorf("action EmergencyReparentShard requires <keyspace/shard> <tablet alias>")
	}

	keyspace, shard, err := topo.ParseKeyspaceShardString(subFlags.Arg(0))
	if err != nil {
		return err
	}
	tabletAlias, err := topo.ParseTabletAliasString(subFlags.Arg(1))
	if err != nil {
		return err
	}
	return wr.EmergencyReparentShard(ctx, keyspace, shard, tabletAlias, *waitSlaveTimeout)
}
