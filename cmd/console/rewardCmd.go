/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package console

import (
	"fmt"
	"os"
	"strings"

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/node"
	sdkgo "github.com/CESSProject/sdk-go"
	"github.com/CESSProject/sdk-go/core/client"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

const (
	reward_cmd       = "reward"
	reward_cmd_use   = "reward"
	reward_cmd_short = "Query reward information"
)

var rewardCmd = &cobra.Command{
	Use:                   reward_cmd_use,
	Short:                 reward_cmd_short,
	Run:                   Command_Reward_Runfunc,
	DisableFlagsInUseLine: true,
}

func init() {
	rootCmd.AddCommand(rewardCmd)
}

// Exit
func Command_Reward_Runfunc(cmd *cobra.Command, args []string) {
	var (
		ok  bool
		err error
		n   = node.New()
	)

	// Build profile instances
	n.Cfg, err = buildConfigFile(cmd, "", 0)
	if err != nil {
		configs.Err(err.Error())
		os.Exit(1)
	}

	//Build client
	cli, err := sdkgo.New(
		configs.Name,
		sdkgo.ConnectRpcAddrs(n.Cfg.GetRpcAddr()),
		sdkgo.ListenPort(n.Cfg.GetServicePort()),
		sdkgo.Workspace(n.Cfg.GetWorkspace()),
		sdkgo.Mnemonic(n.Cfg.GetMnemonic()),
		sdkgo.TransactionTimeout(configs.TimeToWaitEvent),
	)
	if err != nil {
		configs.Err(err.Error())
		os.Exit(1)
	}
	n.Cli, ok = cli.(*client.Cli)
	if !ok {
		configs.Err("Invalid client type")
		os.Exit(1)
	}
	rewardInfo, err := n.Cli.QuaryRewards(n.Cfg.GetPublickey())
	if err != nil {
		configs.Err(err.Error())
		os.Exit(1)
	}
	var total string
	var claimed string
	var available string
	var sep uint8 = 0
	for i := len(rewardInfo.Total) - 1; i >= 0; i-- {
		total = fmt.Sprintf("%c%s", rewardInfo.Total[i], total)
		sep++
		if sep%3 == 0 {
			total = fmt.Sprintf("_%s", total)
		}
	}
	total = strings.TrimPrefix(total, "_")

	sep = 0
	for i := len(rewardInfo.Claimed) - 1; i >= 0; i-- {
		claimed = fmt.Sprintf("%c%s", rewardInfo.Claimed[i], claimed)
		sep++
		if sep%3 == 0 {
			claimed = fmt.Sprintf("_%s", claimed)
		}
	}
	claimed = strings.TrimPrefix(claimed, "_")

	sep = 0
	for i := len(rewardInfo.Available) - 1; i >= 0; i-- {
		available = fmt.Sprintf("%c%s", rewardInfo.Available[i], available)
		sep++
		if sep%3 == 0 {
			available = fmt.Sprintf("_%s", available)
		}
	}
	available = strings.TrimPrefix(available, "_")

	var tableRows = []table.Row{
		{"total reward", total},
		{"claimed reward", claimed},
		{"available reward", available},
	}
	tw := table.NewWriter()
	tw.AppendRows(tableRows)
	fmt.Println(tw.Render())
	os.Exit(0)
}
