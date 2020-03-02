package main

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/Actinium-project/lnd/lnrpc/wtclientrpc"
	"github.com/urfave/cli"
)

// wtclientCommands will return nil for non-wtclientrpc builds.
func wtclientCommands() []cli.Command {
	return []cli.Command{
		{
			Name:     "wtclient",
			Usage:    "Interact with the watchtower client.",
			Category: "Watchtower",
			Subcommands: []cli.Command{
				addTowerCommand,
				removeTowerCommand,
				listTowersCommand,
				getTowerCommand,
				statsCommand,
				policyCommand,
			},
		},
	}
}

// getWtclient initializes a connection to the watchtower client RPC in order to
// interact with it.
func getWtclient(ctx *cli.Context) (wtclientrpc.WatchtowerClientClient, func()) {
	conn := getClientConn(ctx, false)
	cleanUp := func() {
		conn.Close()
	}
	return wtclientrpc.NewWatchtowerClientClient(conn), cleanUp
}

var addTowerCommand = cli.Command{
	Name:  "add",
	Usage: "Register a watchtower to use for future sessions/backups.",
	Description: "If the watchtower has already been registered, then " +
		"this command serves as a way of updating the watchtower " +
		"with new addresses it is reachable over.",
	ArgsUsage: "pubkey@address",
	Action:    actionDecorator(addTower),
}

func addTower(ctx *cli.Context) error {
	// Display the command's help message if the number of arguments/flags
	// is not what we expect.
	if ctx.NArg() != 1 || ctx.NumFlags() > 0 {
		return cli.ShowCommandHelp(ctx, "add")
	}

	parts := strings.Split(ctx.Args().First(), "@")
	if len(parts) != 2 {
		return errors.New("expected tower of format pubkey@address")
	}
	pubKey, err := hex.DecodeString(parts[0])
	if err != nil {
		return fmt.Errorf("invalid public key: %v", err)
	}
	address := parts[1]

	client, cleanUp := getWtclient(ctx)
	defer cleanUp()

	req := &wtclientrpc.AddTowerRequest{
		Pubkey:  pubKey,
		Address: address,
	}
	resp, err := client.AddTower(context.Background(), req)
	if err != nil {
		return err
	}

	printRespJSON(resp)
	return nil
}

var removeTowerCommand = cli.Command{
	Name: "remove",
	Usage: "Remove a watchtower to prevent its use for future " +
		"sessions/backups.",
	Description: "An optional address can be provided to remove, " +
		"indicating that the watchtower is no longer reachable at " +
		"this address. If an address isn't provided, then the " +
		"watchtower will no longer be used for future sessions/backups.",
	ArgsUsage: "pubkey | pubkey@address",
	Action:    actionDecorator(removeTower),
}

func removeTower(ctx *cli.Context) error {
	// Display the command's help message if the number of arguments/flags
	// is not what we expect.
	if ctx.NArg() != 1 || ctx.NumFlags() > 0 {
		return cli.ShowCommandHelp(ctx, "remove")
	}

	// The command can have only one argument, but it can be interpreted in
	// either of the following formats:
	//
	//   pubkey or pubkey@address
	//
	// The hex-encoded public key of the watchtower is always required,
	// while the second is an optional address we'll remove from the
	// watchtower's database record.
	parts := strings.Split(ctx.Args().First(), "@")
	if len(parts) > 2 {
		return errors.New("expected tower of format pubkey@address")
	}
	pubKey, err := hex.DecodeString(parts[0])
	if err != nil {
		return fmt.Errorf("invalid public key: %v", err)
	}
	var address string
	if len(parts) == 2 {
		address = parts[1]
	}

	client, cleanUp := getWtclient(ctx)
	defer cleanUp()

	req := &wtclientrpc.RemoveTowerRequest{
		Pubkey:  pubKey,
		Address: address,
	}
	resp, err := client.RemoveTower(context.Background(), req)
	if err != nil {
		return err
	}

	printRespJSON(resp)
	return nil
}

var listTowersCommand = cli.Command{
	Name:  "towers",
	Usage: "Display information about all registered watchtowers.",
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name: "include_sessions",
			Usage: "include sessions with the watchtower in the " +
				"response",
		},
	},
	Action: actionDecorator(listTowers),
}

func listTowers(ctx *cli.Context) error {
	// Display the command's help message if the number of arguments/flags
	// is not what we expect.
	if ctx.NArg() > 0 || ctx.NumFlags() > 1 {
		return cli.ShowCommandHelp(ctx, "towers")
	}

	client, cleanUp := getWtclient(ctx)
	defer cleanUp()

	req := &wtclientrpc.ListTowersRequest{
		IncludeSessions: ctx.Bool("include_sessions"),
	}
	resp, err := client.ListTowers(context.Background(), req)
	if err != nil {
		return err
	}

	printRespJSON(resp)

	return nil
}

var getTowerCommand = cli.Command{
	Name:      "tower",
	Usage:     "Display information about a specific registered watchtower.",
	ArgsUsage: "pubkey",
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name: "include_sessions",
			Usage: "include sessions with the watchtower in the " +
				"response",
		},
	},
	Action: actionDecorator(getTower),
}

func getTower(ctx *cli.Context) error {
	// Display the command's help message if the number of arguments/flags
	// is not what we expect.
	if ctx.NArg() != 1 || ctx.NumFlags() > 1 {
		return cli.ShowCommandHelp(ctx, "tower")
	}

	// The command only has one argument, which we expect to be the
	// hex-encoded public key of the watchtower we'll display information
	// about.
	pubKey, err := hex.DecodeString(ctx.Args().Get(0))
	if err != nil {
		return fmt.Errorf("invalid public key: %v", err)
	}

	client, cleanUp := getWtclient(ctx)
	defer cleanUp()

	req := &wtclientrpc.GetTowerInfoRequest{
		Pubkey:          pubKey,
		IncludeSessions: ctx.Bool("include_sessions"),
	}
	resp, err := client.GetTowerInfo(context.Background(), req)
	if err != nil {
		return err
	}

	printRespJSON(resp)
	return nil
}

var statsCommand = cli.Command{
	Name:   "stats",
	Usage:  "Display the session stats of the watchtower client.",
	Action: actionDecorator(stats),
}

func stats(ctx *cli.Context) error {
	// Display the command's help message if the number of arguments/flags
	// is not what we expect.
	if ctx.NArg() > 0 || ctx.NumFlags() > 0 {
		return cli.ShowCommandHelp(ctx, "stats")
	}

	client, cleanUp := getWtclient(ctx)
	defer cleanUp()

	req := &wtclientrpc.StatsRequest{}
	resp, err := client.Stats(context.Background(), req)
	if err != nil {
		return err
	}

	printRespJSON(resp)
	return nil
}

var policyCommand = cli.Command{
	Name:   "policy",
	Usage:  "Display the active watchtower client policy configuration.",
	Action: actionDecorator(policy),
}

func policy(ctx *cli.Context) error {
	// Display the command's help message if the number of arguments/flags
	// is not what we expect.
	if ctx.NArg() > 0 || ctx.NumFlags() > 0 {
		return cli.ShowCommandHelp(ctx, "policy")
	}

	client, cleanUp := getWtclient(ctx)
	defer cleanUp()

	req := &wtclientrpc.PolicyRequest{}
	resp, err := client.Policy(context.Background(), req)
	if err != nil {
		return err
	}

	printRespJSON(resp)
	return nil
}
